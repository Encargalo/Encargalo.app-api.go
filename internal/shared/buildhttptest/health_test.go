package buildhttptest

import (
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/labstack/echo/v4"
)

type ControllerCase struct {
	Req *http.Request
	Res *httptest.ResponseRecorder
	Ctx echo.Context
}

func SetupControllerCase(url, method string, body io.Reader) ControllerCase {
	e := echo.New()
	req := httptest.NewRequest(method, url, body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	res := httptest.NewRecorder()
	ctx := e.NewContext(req, res)

	return ControllerCase{req, res, ctx}
}
