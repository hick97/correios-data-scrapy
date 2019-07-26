package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLocalities(t *testing.T) {
	router := localitiesHandler()

	server := httptest.NewServer(router)
	defer server.Close()

	/* GET Localities*/
/* 
	req, err := http.NewRequest(http.MethodGet, server.URL+"/v1/localities/PB,CE,AL", nil)
	assert.NoError(t, err)

	res, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, res.StatusCode)

	/* GET Localities by UF */
	/*
	req, err = http.NewRequest(http.MethodGet, server.URL+"/v1/localities/AC", nil)
	assert.NoError(t, err)

	res, err = http.DefaultClient.Do(req)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, res.StatusCode)

	/* GET Localities with invalid UF */

	req, err := http.NewRequest(http.MethodGet, server.URL+"/v1/localities/JI", nil)
	assert.NoError(t, err)

	res, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusNotFound, res.StatusCode)

	/* GET Data when params have space */

	req, err = http.NewRequest(http.MethodGet, server.URL+"/v1/localities/PB , CE,AL ", nil)
	assert.NoError(t, err)

	res, err = http.DefaultClient.Do(req)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, res.StatusCode)

	/* GET more than five UFs */

	req, err = http.NewRequest(http.MethodGet, server.URL+"/v1/localities/PB,CE,SP,MT,RJ,PE", nil)
	assert.NoError(t, err)

	res, err = http.DefaultClient.Do(req)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, res.StatusCode)

	/* Database */

	/* POST */

	var localities = []Locality{
		Locality{
			Name:     "Acrelândia",
			CEPRange: "69945-000 a 69949-999",
		},
		Locality{
			Name:     "Acrelândia",
			CEPRange: "69945-000 a 69949-999",
		},
		Locality{
			Name:     "Acrelândia",
			CEPRange: "69945-000 a 69949-999",
		},
		Locality{
			Name:     "Acrelândia",
			CEPRange: "69945-000 a 69949-999",
		},
	}

	federativeUnit := FederativeUnit{
		Name:       "AC",
		Localities: localities,
	}

	err = UFPersistence(federativeUnit)
	assert.NoError(t, err)

	/* GET */
	UF := "AC"

	result, err := checkIfUFAlreadyExists(UF)
	assert.NoError(t, err)

	assert.NotEqual(t, result, nil)
	
}
