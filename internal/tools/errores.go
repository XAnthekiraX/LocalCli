package tools

// Códigos internos de error ([[backend/05-quality/ERRORS]] §3).
const (
	CodToolUnknown        = "E_TOOL_UNKNOWN"
	CodToolNotAllowed     = "E_TOOL_NOT_ALLOWED"
	CodBadArgs            = "E_BAD_ARGS"
	CodPathOutside        = "E_PATH_OUTSIDE"
	CodPathExists         = "E_PATH_EXISTS"
	CodNeedsConfirm       = "E_NEEDS_CONFIRM"
	CodNeedsApproval      = "E_NEEDS_APPROVAL"
	CodApprovalDeclined   = "E_APPROVAL_DECLINED"
	CodCmdNotWhitelisted  = "E_CMD_NOT_WHITELISTED"
	CodCmdTimeout         = "E_CMD_TIMEOUT"
	CodCmdOutputTruncated = "E_CMD_OUTPUT_TRUNCATED"
	CodNoLandlock         = "E_NO_LANDLOCK"
	// Flujos ([[backend/05-quality/ERRORS]] §5).
	CodFlowUnknown    = "E_FLOW_UNKNOWN"
	CodFlowNotAllowed = "E_FLOW_NOT_ALLOWED"
)

// FaltanCampos comprueba que los argumentos traigan los campos obligatorios del
// esquema y nombra el primero que falta.
func FaltanCampos(e *Esquema, argumentos map[string]any) string {
	for _, requerido := range e.Requeridos {
		if _, ok := argumentos[requerido]; !ok {
			return requerido
		}
	}
	return ""
}
