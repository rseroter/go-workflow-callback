package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strings"

	metadata "cloud.google.com/go/compute/metadata"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Template struct {
	Templates *template.Template
}

type workflowdata struct {
	CallbackUrl string
	LoanId      string
}

// OAuth2 JSON struct
type OAuth2TokenInfo struct {
	// defining struct variables
	Token      string `json:"access_token"`
	TokenType  string `json:"token_type"`
	Expiration uint32 `json:"expires_in"`
}

// TemplateRenderer is a custom html/template renderer for Echo framework
type TemplateRenderer struct {
	templates *template.Template
}

//implement echo interface
func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.Templates.ExecuteTemplate(w, name, data)
}

func main() {

	fmt.Println("Started up ...")

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	t := &Template{
		Templates: template.Must(template.ParseGlob("web/home.html")),
	}

	e.Renderer = t
	e.GET("/", func(c echo.Context) error {

		//load up object with querystring parameters
		wf := workflowdata{LoanId: c.QueryParam("loanid"), CallbackUrl: c.QueryParam("callbackurl")}

		//passing in the template name (not file name)
		return c.Render(http.StatusOK, "home", wf)
	})

	//respond to POST requests and send message to callback URL
	e.POST("/ack", func(c echo.Context) error {

		loanid := c.FormValue("loanid")
		fmt.Println(loanid)
		callbackurl := c.FormValue("callbackurl")

		fmt.Println("Sending workflow callback to " + callbackurl)

		wf := workflowdata{LoanId: loanid, CallbackUrl: callbackurl}

		// Fetch an OAuth2 access token from the metadata server
		oauthToken, errAuth := metadata.Get("instance/service-accounts/default/token")
		if errAuth != nil {
			fmt.Println(errAuth)
		}

		//load up oauth token
		data := OAuth2TokenInfo{}
		errJson := json.Unmarshal([]byte(oauthToken), &data)
		if errJson != nil {
			fmt.Println(errJson.Error())
		}
		fmt.Printf("OAuth2 token: %s", data.Token)

		//setup callback request
		workflowReq, errWorkflowReq := http.NewRequest("POST", callbackurl, strings.NewReader("{}"))
		if errWorkflowReq != nil {
			fmt.Println(errWorkflowReq.Error())
		}

		//add oauth header
		workflowReq.Header.Add("authorization", "Bearer "+data.Token)
		workflowReq.Header.Add("accept", "application/json")
		workflowReq.Header.Add("content-type", "application/json")

		//inboke callback url
		client := &http.Client{}
		workflowResp, workflowErr := client.Do(workflowReq)

		if workflowErr != nil {

			fmt.Printf("Error making callback request: %s\n", workflowErr)
		}
		fmt.Printf("Status code: %d", workflowResp.StatusCode)

		return c.Render(http.StatusOK, "home", wf)
	})

	//simple startup
	e.Logger.Fatal(e.Start(":8080"))
}
