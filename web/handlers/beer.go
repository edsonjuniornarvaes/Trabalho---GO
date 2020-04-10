package handlers

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"

	"github.com/codegangsta/negroni"
	"github.com/eminetto/pos-web-go/core/beer"
	"github.com/gorilla/mux"
)

//a função recebe como terceiro parâmetro a interface
//ou seja, ela pode receber qualquer coisa que implemente a interface
//isso é muito útil para escrevermos testes, ou podermos substituir toda a
//implementação da regra de negócios
func MakeBeerHandlers(r *mux.Router, n *negroni.Negroni, service beer.UseCase) {
	r.Handle("/v1/beer", n.With(
		negroni.Wrap(getAllBeer(service)),
	)).Methods("GET", "OPTIONS")

	r.Handle("/v1/beer/{id}", n.With(
		negroni.Wrap(getBeer(service)),
	)).Methods("GET", "OPTIONS")

	r.Handle("/v1/beer", n.With(
		negroni.Wrap(storeBeer(service)),
	)).Methods("POST", "OPTIONS")

	r.Handle("/v1/beer/{id}", n.With(
		negroni.Wrap(updateBeer(service)),
	)).Methods("PUT", "OPTIONS")

	r.Handle("/v1/beer/{id}", n.With(
		negroni.Wrap(removeBeer(service)),
	)).Methods("DELETE", "OPTIONS")
}

/*
Para testar:
curl http://localhost:4000/v1/beer
*/
func getAllBeer(service beer.UseCase) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//analisa o que o usuário requisitou via headers
		switch r.Header.Get("Accept") {
		case "application/json":
			getAllBeerJSON(w, service)
		default:
			getAllBeerHTML(w, service)
		}

	})
}

func getAllBeerHTML(w http.ResponseWriter, service beer.UseCase) {
	ts, err := template.ParseFiles(
		"./web/templates/header.html",
		"./web/templates/index.html",
		"./web/templates/footer.html")
	if err != nil {
		http.Error(w, "Error parsing "+err.Error(), http.StatusInternalServerError)
		return
	}
	all, err := service.GetAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := struct {
		Title string
		Beers []*beer.Beer
	}{
		Title: "Beers",
		Beers: all,
	}
	err = ts.Lookup("index.html").ExecuteTemplate(w, "index", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func getAllBeerJSON(w http.ResponseWriter, service beer.UseCase) {
	all, err := service.GetAll()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(formatJSONError(err.Error()))
		return
	}
	//vamos converter o resultado em JSON e gerar a resposta
	err = json.NewEncoder(w).Encode(all)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(formatJSONError("Erro convertendo em JSON"))
		return
	}
}

/*
Para testar:
curl http://localhost:4000/v1/beer/1
*/
func getBeer(service beer.UseCase) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//vamos pegar o ID da URL
		//na definição do protocolo http, os parâmetros são enviados no formato de texto
		//por isso precisamos converter em int
		vars := mux.Vars(r)
		id, err := strconv.ParseInt(vars["id"], 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write(formatJSONError(err.Error()))
			return
		}
		b, err := service.Get(id)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			w.Write(formatJSONError(err.Error()))
			return
		}
		//vamos converter o resultado em JSON e gerar a resposta
		err = json.NewEncoder(w).Encode(b)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(formatJSONError("Erro convertendo em JSON"))
			return
		}
	})
}

/*
Para testar:
curl -X "POST" "http://localhost:4000/v1/beer" \
     -H 'Accept: application/json' \
     -H 'Content-Type: application/json' \
     -d $'{
  "name": "Skol",
  "type": 1,
  "style":2
}'
*/
func storeBeer(service beer.UseCase) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//vamos pegar os dados enviados pelo usuário via body
		var b beer.Beer
		err := json.NewDecoder(r.Body).Decode(&b)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write(formatJSONError(err.Error()))
			return
		}

		err = b.Validate()
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write(formatJSONError(err.Error()))
			return
		}

		err = service.Store(&b)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(formatJSONError(err.Error()))
			return
		}
		w.WriteHeader(http.StatusCreated)
	})
}

/*
Para testar:
curl -X "PUT" "http://localhost:4000/v1/beer/2" \
     -H 'Accept: application/json' \
     -H 'Content-Type: application/json' \
     -d $'{
  "name": "Alterada",
  "type": 3,
  "style":1
}'
*/
func updateBeer(service beer.UseCase) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// recebe o id que será alterado
		data := mux.Vars(r)

		id, err := strconv.ParseInt(data["id"], 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write(formatJSONError(err.Error()))

			return
		}

		oldBeer, err := service.Get(id)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			w.Write(formatJSONError(err.Error()))

			return
		}

		// Recebe os dados do Put
		var newBeer beer.Beer
		err = json.NewDecoder(r.Body).Decode(&newBeer)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write(formatJSONError(err.Error()))

			return
		}

		err = newBeer.Validate()
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write(formatJSONError(err.Error()))
			return
		}

		oldBeer.Name = newBeer.Name
		oldBeer.Style = newBeer.Style
		oldBeer.Type = newBeer.Type

		err = service.Update(oldBeer)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(formatJSONError(err.Error()))

			return
		}

		w.WriteHeader(http.StatusNoContent)
	})
}

/*
Para testar:
curl -X "DELETE" "http://localhost:4000/v1/beer/2" \
     -H 'Accept: application/json' \
     -H 'Content-Type: application/json'
*/
func removeBeer(service beer.UseCase) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Recebe o id
		data := mux.Vars(r)
		id, err := strconv.ParseInt(data["id"], 10, 64)

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write(formatJSONError(err.Error()))

			return
		}

		err = service.Remove(id)

		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			w.Write(formatJSONError(err.Error()))

			return
		}

		w.WriteHeader(http.StatusNoContent)
	})
}
