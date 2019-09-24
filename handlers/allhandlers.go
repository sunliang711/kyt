package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/rs/cors"

	"github.com/sunliang711/kyt/models"
	"github.com/sunliang711/kyt/utils"
)

type HandlerObj struct {
	H       http.HandlerFunc
	Path    string
	Methods []string
	Usage   string
}

var allHandlers = []*HandlerObj{
	&HandlerObj{searchType, "/search_type", []string{"GET"}, ""},
	&HandlerObj{addressIdentity, "/address_identity", []string{"GET"}, ""},
	&HandlerObj{addressRadar, "/address_radar", []string{"GET"}, ""},
	&HandlerObj{addressSourceType, "/address_sourcetype", []string{"GET"}, ""},
	&HandlerObj{addressRiskLevel, "/address_risklevel", []string{"GET"}, ""},
	&HandlerObj{addressBadTxList, "/address_badtxlist", []string{"GET"}, ""},
	&HandlerObj{addressBadTxGraph, "/address_badtx_graph", []string{"GET"}, ""},
	&HandlerObj{blockTxList, "/block_txlist", []string{"GET"}, ""},
	&HandlerObj{transactionIdentity, "/transaction_identity", []string{"GET"}, ""},
	&HandlerObj{transactionTxList, "/transaction_txlist", []string{"GET"}, ""},
	&HandlerObj{transactionGraph, "/transaction_graph", []string{"GET"}, ""},
	&HandlerObj{blockTransactionGraph, "/block_transaction_graph", []string{"GET"}, ""},
}

func Router(enableCors bool) http.Handler {
	rt := mux.NewRouter()

	for _, h := range allHandlers {
		rt.Handle(h.Path, h.H).Methods(h.Methods...)
	}

	if enableCors {
		return cors.Default().Handler(rt)
	} else {
		return rt
	}
}

type resp struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

//"symbol='BTC'
//querystr='xxx'"

func searchType(w http.ResponseWriter, req *http.Request) {
	query := req.URL.Query()
	queryType := query.Get("symbol")
	queryValue := query.Get("querystr")
	switch queryType {
	case "BTC":
		fmt.Printf("query value:%v\n", queryValue)
		if len(queryValue) == 0 {
			utils.JsonResponse(resp{1, "Need query parameter", nil}, w)
			return
		}

		rows, err := models.DB.Query("select address from addresses where address = ? ", queryValue)
		if err != nil {
			fmt.Println("query from addresses by address error")
			utils.JsonResponse(resp{1, "internel db error", nil}, w)
			return
		}
		defer rows.Close()

		if rows.Next() {
			//exist
			log.Println("exist in addresses table")
			utils.JsonResponse(resp{0, "OK", utils.TypeAddress}, w)
			return
		} else {
			queryValueInt, err := strconv.Atoi(queryValue)
			if err == nil {
				fmt.Printf("queryValueInt: %d\n", queryValueInt)
				rows, err = models.DB.Query("select * from blockinfo where blockID = ? ", queryValueInt)
				if err != nil {
					fmt.Printf("query from blockinfo by blockID error")
					utils.JsonResponse(resp{1, "internel db error", nil}, w)
					return
				}
				if rows.Next() {
					log.Println("exist in blockinfo table")
					utils.JsonResponse(resp{0, "OK", utils.TypeBlockID}, w)
					return
				}
			}
			rows, err = models.DB.Query("select * from blockinfo where blockhash = ?", queryValue)
			if err != nil {
				fmt.Printf("query from blockinfo by blockhash error")
				utils.JsonResponse(resp{1, "internel db error", nil}, w)
				return
			}
			if rows.Next() {
				log.Println("exist in blockinfo table")
				utils.JsonResponse(resp{0, "OK", utils.TypeBlockHash}, w)
				return
			} else {
				rows, err = models.DB.Query("select * from txhash where txhash = ? ", queryValue)
				if err != nil {
					fmt.Printf("query from txhash by hash error")
					utils.JsonResponse(resp{1, "internel db error", nil}, w)
					return
				}
				if rows.Next() {
					log.Println("exist in txhash table")
					utils.JsonResponse(resp{0, "OK", utils.TypeTx}, w)
					return
				} else {
					log.Println("exist in other")
					utils.JsonResponse(resp{0, "OK", utils.TypeOther}, w)
					return
				}
			}

		}

	default:
		utils.JsonResponse(resp{1, "symbol not support", nil}, w)
		return
	}
}
