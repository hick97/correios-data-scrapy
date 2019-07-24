package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
	"github.com/gin-gonic/gin"
)

// UF Schema
type FederativeUnit struct {
	Name       string     `json:"uf_name" bson:"uf_name"`
	Localities []Locality `json:"localities" bson:"localities"`
}

// Locality Schema
type Locality struct {
	Name     string `json:"locality_name" bson:"locality_name"`
	CEPRange string `json:"cep_range" bson:"cep_range"`
}

//Main function
func main() {
	router := localitiesHandler()

	router.Run()
}

// Main handler
func localitiesHandler() *gin.Engine {
	router := gin.Default()

	//Routes

	v1 := router.Group("/v1")

	v1.GET("/localities", getLocalities)

	v1.GET("/localities/:uf", getLocalitiesByUF)

	return router
}

//Controllers
func getLocalities(c *gin.Context) {
	c.AbortWithStatus(http.StatusOK)
}

func getLocalitiesByUF(c *gin.Context) {
	UF := c.Param("uf")
	var domHTML string
	Results := []Locality{}

	fmt.Println("Startando contexto..")

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	fmt.Println("Executando tarefas...")

	chromeDpTask := chromedp.Tasks{
		chromedp.Navigate("http://www.buscacep.correios.com.br/sistemas/buscacep/buscaFaixaCep.cfm"),
		chromedp.WaitVisible("#Geral select"),
		chromedp.SetAttributeValue(`#Geral select option[value="`+UF+`"]`, "selected", "true"),
		chromedp.WaitSelected(`#Geral select option[value="` + UF + `"]`),

		chromedp.Click(`#Geral input[value="Buscar"]`),
		chromedp.WaitVisible(`table[class*="tmptabela"]:nth-of-type(2)`),
		chromedp.OuterHTML(`div[class*="ctrlcontent"]`, &domHTML),
	}

	err := chromedp.Run(ctx, chromeDpTask)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
	}

	fmt.Println("Executei!")

	//Receive the HTML from panel element. This element has the result table
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(domHTML))
	if err != nil {

		c.AbortWithError(http.StatusInternalServerError, err)
	}

	Results, err = extractTableData(doc, Results, false)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
	}

	haveNextButton := doc.Find(`form[name="Proxima"]`)
	for haveNextButton.Size() > 0 {
		doc, err = whileHaveNext(ctx)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
		}

		haveNextButton = doc.Find(`form[name="Proxima"]`)

		Results, err = extractTableData(doc, Results, true)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
		}
	}

	c.JSON(http.StatusOK, Results)
}

// Support functions
func extractTableData(doc *goquery.Document, Results []Locality, isNextButton bool) ([]Locality, error) {

	tableIndex := "1"
	if !isNextButton {
		tableIndex = "2"
	}

	if doc.Find(`table[class*="tmptabela"]:nth-of-type(`+tableIndex+`)`).Size() > 0 {

		doc.Find(`table[class*="tmptabela"]:nth-of-type(` + tableIndex + `) tbody tr`).Each(func(i int, selection *goquery.Selection) {

			// For each locality finded, get your name and CEP range
			locality := Locality{}
			locality.Name = selection.Find("td:nth-child(1)").Text()
			locality.CEPRange = selection.Find("td:nth-child(2)").Text()

			//the 1st row (header row) has empty fields, so is ignored
			if locality.Name != "" {
				Results = append(Results, locality)
			}

		})
	}

	return Results, nil
}

// While have 'next' button, make this actions
func whileHaveNext(ctx context.Context) (*goquery.Document, error) {
	var domHTML string

	chromeDpTask2 := chromedp.Tasks{
		chromedp.WaitVisible(`form[name="Proxima"]`),
		chromedp.Click(`div[class*="ctrlcontent"] div[style="float:left"]:nth-of-type(2)`),
		chromedp.WaitVisible(`table[class*="tmptabela"]`),
		chromedp.OuterHTML(`div[class*="ctrlcontent"]`, &domHTML),
	}

	err := chromedp.Run(ctx, chromeDpTask2)
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(domHTML))
	if err != nil {
		return nil, err
	}

	return doc, nil
}
