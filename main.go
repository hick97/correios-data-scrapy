package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

//UF validator
var UFValidator map[string]string

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

	s := &http.Server{
		Addr:         ":8080",
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	s.ListenAndServe()

	router.Run()

}

// Main handler
func localitiesHandler() *gin.Engine {
	router := gin.Default()

	//Routes

	v1 := router.Group("/v1")

	v1.GET("/localities/:ufs", getLocalities)

	return router
}

//Controllers

func getLocalities(c *gin.Context) {
	params := c.Param("ufs")
	options := strings.Split(params, ",")

	if len(options) > 5 {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	resultCh := make(chan []Locality)
	var answers []FederativeUnit

	// create chrome instance
	ctx, cancel := chromedp.NewContext(
		context.Background(),
		chromedp.WithLogf(log.Printf),
	)
	defer cancel()

	// create a timeout
	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	for _, value := range options {
		go func() {
			value = strings.Trim(value, " ")
			isValid, err := UFIsValid(value)

			if !isValid {
				c.AbortWithError(http.StatusNotFound, err)
				close(resultCh)
				return
			}

			err = crawlerExecution(ctx, value, resultCh)
			if err != nil {
				c.AbortWithError(http.StatusInternalServerError, err)
				return
			}
		}()

		localities := <-resultCh

		finalResult := FederativeUnit{
			Name:       value,
			Localities: localities,
		}

		answers = append(answers, finalResult)

		fmt.Println(finalResult)
	}

	//Output genertion
	jsonlResponse(answers)

	jsonResponse(answers)

	c.AbortWithStatus(http.StatusOK)
}
func jsonlResponse(answers []FederativeUnit) error {

	file, err := os.Create("result.jsonl")
	if err != nil {
		return err
	}
	defer file.Close()

	w := bufio.NewWriter(file)
	for _, line := range answers {
		bytes, err := json.Marshal(line)
		if err != nil {
			log.Fatal(err)
		}

		if len(line.Localities) > 0 {
			bytes_to_string := string(bytes)
			fmt.Fprintln(w, bytes_to_string)
		}

	}
	return w.Flush()
}

func jsonResponse(answers []FederativeUnit) {
	json, err := json.Marshal(answers)
	if err != nil {
		log.Fatal(err)
	}

	_ = ioutil.WriteFile("result.json", json, 0644)
}

func crawlerExecution(ctx context.Context, uf string, op chan []Locality) error {

	UF := uf
	var domHTML string
	Results := []Locality{}

	chromeDpTask := chromedp.Tasks{
		chromedp.Navigate("http://www.buscacep.correios.com.br/sistemas/buscacep/buscaFaixaCep.cfm"),
		chromedp.WaitVisible("#Geral select"),
		chromedp.SetAttributeValue(`#Geral select option[value="`+UF+`"]`, "selected", "true"),
		chromedp.WaitSelected(`#Geral select option[value="` + UF + `"]`),
		chromedp.Click(`#Geral input[value="Buscar"]`),
		chromedp.Sleep(1 * time.Second),
		//chromedp.WaitVisible(`table[class*="tmptabela"]:nth-of-type(2)`),
		chromedp.OuterHTML(`div[class*="ctrlcontent"]`, &domHTML),
	}

	err := chromedp.Run(ctx, chromeDpTask)
	if err != nil {
		return err
	}

	//Receive the HTML from panel element. This element has the result table
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(domHTML))
	if err != nil {
		return err
	}

	Results, err = extractTableData(doc, Results, false)
	if err != nil {
		return err
	}

	haveNextButton := doc.Find(`form[name="Proxima"]`)

	for haveNextButton.Size() > 0 {
		//fmt.Println("- Encontrei nova página")
		doc, err = whileHaveNext(ctx)
		if err != nil {
			return err
		}

		//fmt.Println("- Passei do whilehve next")

		haveNextButton = doc.Find(`form[name="Proxima"]`)

		Results, err = extractTableData(doc, Results, true)
		if err != nil {
			return err
		}
		//fmt.Println("- Extrai dados da pagina nova")
	}

	time.Sleep(time.Millisecond * 10)

	op <- Results

	return nil

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
		//chromedp.Sleep(500 * time.Millisecond),
		chromedp.WaitVisible(`form[name="Proxima"]`),
		chromedp.Click(`div[class*="ctrlcontent"] div[style="float:left"]:nth-of-type(2)`),
		chromedp.Sleep(1 * time.Second),
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

// Support functions

// Validating UF param
func UFIsValid(UF string) (bool, error) {
	UFValidator := map[string]string{
		"AC": "AC",
		"AL": "AL",
		"AP": "AP",
		"AM": "AM",
		"BA": "BA",
		"CE": "CE",
		"DF": "DF",
		"ES": "ES",
		"GO": "GO",
		"MA": "MA",
		"MT": "MT",
		"MS": "MS",
		"MG": "MG",
		"PA": "PA",
		"PB": "PB",
		"PR": "PR",
		"PE": "PE",
		"PI": "PI",
		"RJ": "RJ",
		"RN": "RN",
		"RS": "RS",
		"RO": "RO",
		"RR": "RR",
		"SC": "SC",
		"SP": "SP",
		"SE": "SE",
		"TO": "TO",
	}

	_, ok := UFValidator[UF]

	if !ok {
		return false, fmt.Errorf("Invalid UF")
	}

	return true, nil
}

// Persist UF's data
func UFPersistence(UFData FederativeUnit) error {

	session, err := mgo.Dial("172.21.0.2")
	if err != nil {
		return err
	}

	if err := session.DB("cep-db").C("federativeUnit").Insert(UFData); err != nil {
		return err
	}

	return nil

}

func checkIfUFAlreadyExists(UF string) ([]map[string]interface{}, error) {
	session, err := mgo.Dial("172.21.0.2")
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}

	err = session.DB("cep-db").C("federativeUnit").Find(bson.M{"uf_name": UF}).All(&result)

	if err != nil {
		if err == mgo.ErrNotFound {
			return nil, err
		}
		return nil, err
	}
	return result, nil
}
