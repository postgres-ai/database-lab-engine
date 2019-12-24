package srv

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"../log"

	"github.com/gorilla/mux"
)

// Router traversal in order to get a list of all routes.
func getHelpRoutes(router *mux.Router) ([]Route, error) {
	routes := make([]Route, 0)
	err := router.Walk(func(route *mux.Route, router *mux.Router,
		ancestors []*mux.Route) error {
		pathTemplate, err := route.GetPathTemplate()
		if err != nil {
			return err
		}

		methods, err := route.GetMethods()
		if err != nil {
			return err
		}

		routes = append(routes, Route{
			Route:   pathTemplate,
			Methods: methods,
		})

		return nil
	})

	return routes, err
}

// Respond with JSON.
func writeJson(w http.ResponseWriter, v interface{}) (int, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Err(err)
		return 0, err
	}

	n, err := w.Write(b)
	if err != nil {
		log.Err(err)
		return 0, err
	}

	log.Dbg("Response:", v)

	return n, nil
}

// Read JSON from request.
func readJson(r *http.Request, v interface{}) error {
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Err(err, "\n", string(reqBody))
		return err
	}

	log.Dbg("Request:", string(reqBody))

	err = json.Unmarshal(reqBody, v)
	if err != nil {
		log.Err(err, "\n", string(reqBody))
		return err
	}

	log.Dbg("Request:", v)

	return nil
}
