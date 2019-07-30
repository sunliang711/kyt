package utils

import (
	"encoding/json"
	"github.com/sirupsen/logrus"
	"net/http"
)

func JsonResponse(data interface{}, w http.ResponseWriter) {
	b, err := json.Marshal(data)
	if err != nil {
		logrus.Error("json.Marshal:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(b)

}
