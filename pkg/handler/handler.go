package handler

import (
	"database/sql"
	"net/http"
	"regexp"
	"strings"

	"github.com/Essen-Labs/bloom-be/pkg/config"
	"github.com/Essen-Labs/bloom-be/pkg/constant"
	"github.com/Essen-Labs/bloom-be/pkg/util"
	"github.com/Essen-Labs/bloom-be/translation"
	"github.com/dwarvesf/gerr"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// Handler for app
type Handler struct {
	log        gerr.Log
	cfg        config.Config
	translator translation.Helper
	db         *sql.DB
}

// NewHandler make handler
func NewHandler(cfg config.Config, l gerr.Log, th translation.Helper, db *sql.DB) *Handler {
	return &Handler{
		log:        l,
		cfg:        cfg,
		translator: th,
		db:         db,
	}
}

func (h *Handler) handleError(c *gin.Context, err error) {
	if err == nil {
		return
	}
	locale := c.GetString(constant.LanguageKey)
	tr := h.translator.GetTranslator(locale)

	var parsedErr gerr.Error
	switch arg := err.(type) {
	case validator.ValidationErrors:
		ds := []gerr.Error{}
		childrens := []gerr.CombinedItem{}
		for _, currErr := range arg {
			msg := currErr.Translate(tr)
			targetStr := util.RemoveFirstElementBySeparator(currErr.Namespace(), ".")
			targets := makeKeysFromTarget(targetStr)
			targetCombined := strings.Join(targets, ".")
			ds = append(ds, gerr.E(msg, gerr.Target(targetCombined)))

			childrens = append(childrens, gerr.CombinedItem{
				Keys:    targets,
				Message: msg,
			})
		}

		badRequest := "bad request"
		msg, err := tr.T(badRequest)
		if err != nil {
			msg = badRequest
		}
		rs := gerr.CombinedE(
			http.StatusBadRequest,
			msg,
			childrens,
		)
		parsedErr = *rs.ToError()

	case gerr.Error:
		parsedErr = arg

	case *gerr.Error:
		parsedErr = *arg

	case error:
		str := arg.Error()
		if str == "EOF" {
			parsedErr = gerr.E("bad request", http.StatusBadRequest)
			break
		}
		parsedErr = gerr.E(arg.Error(), http.StatusInternalServerError)
	}

	// log data to console
	logDataRaw, ok := c.Get(constant.LogDataKey)
	traceID := ""
	if ok {
		if ld, parsed := logDataRaw.(gerr.LogInfo); parsed {
			traceID = ld.GetTraceID()
		}
	}
	h.log.Log(logDataRaw, parsedErr) //nolint:errcheck // Ignore unused function warning

	c.AbortWithStatusJSON(parsedErr.StatusCode(), parsedErr.ToResponseError(traceID))
}

func makeKeysFromTarget(target string) []string {
	keys := strings.Split(target, ".")
	rs := []string{}
	reg, _ := regexp.Compile("(.+)\\[(.+)\\]") //nolint
	for idx := range keys {
		k := keys[idx]
		itms := reg.FindStringSubmatch(k)
		if len(itms) <= 0 {
			rs = append(rs, k)
			continue
		}
		rs = append(rs, itms[1:]...)

	}
	return rs
}

// Healthz godoc
// @Summary Health check
// @Description Check if the service is running
// @Tags health
// @Accept  json
// @Produce  json
// @Success 200 {string} string "OK"
// @Router /healthz [get]
func (h *Handler) Healthz(c *gin.Context) {
	c.Header("Content-Type", "text/plain")
	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.Write([]byte("OK")) //nolint
}
