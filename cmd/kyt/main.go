package main

import (
	"fmt"
	gh "github.com/gorilla/handlers"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/sunliang711/kyt/handlers"
	"github.com/sunliang711/kyt/models"
	"github.com/sunliang711/kyt/utils"
	"io"
	"log"
	"net/http"
	"os"
)

//https://org.modao.cc/app/d05bb86439ef70dcdef4ac606045b93d#screen=s06EC007C3A1561965126814

func main() {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatal(err)
	}

	models.InitDB(viper.GetString("dsn"))

	rt := handlers.Router(viper.GetBool("cor"))

	logrus.SetLevel(utils.LogLevel(viper.GetString("loglevel")))
	//set logger
	var w io.Writer
	if len(viper.GetString("output")) == 0 {
		w = os.Stdout
	} else {
		f, err := os.OpenFile(viper.GetString("output"), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			panic(err)
		}
		w = io.MultiWriter(f, os.Stdout)
	}
	addr := fmt.Sprintf(":%d", viper.GetInt("port"))
	log.Printf("Rest server listening on %s", addr)
	log.SetOutput(w)

	//run
	http.ListenAndServe(addr, gh.LoggingHandler(w, gh.CompressHandler(rt)))

}
