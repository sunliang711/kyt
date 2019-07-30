package handlers

import (
	"database/sql"
	"fmt"
	"github.com/sunliang711/kyt/models"
	"github.com/sunliang711/kyt/utils"
	"log"
	"net/http"
	"strconv"
)

type tx struct {
	ID  int64  `json:"id"`
	Tag string `json:"tag"`
}

func blockTxList(w http.ResponseWriter, req *http.Request) {
	query := req.URL.Query()
	var (
		err         error
		blockheight int
		rows        *sql.Rows
		txID        int64
		tag         string
	)
	blockid := query.Get("blockid")
	blockhash := query.Get("blockhash")
	if len(blockid) != 0 {
		blockheight, err = strconv.Atoi(blockid)
		if err != nil {
			utils.JsonResponse(resp{1, "block id format error", nil}, w)
			return
		}
		log.Printf("Got blockheight: %v\n", blockheight)

	} else if len(blockhash) != 0 {
		log.Printf("Got blockhash: %v", blockhash)
		//如果是blockhash,要通过blockhash拿到blockid
		rows, err := models.DB.Query("select blockID from blockinfo where blockhash=?", blockhash)
		if err != nil {
			utils.JsonResponse(resp{1, fmt.Sprintf("SQL error: select blockID from blockinfo where blockhash = %v", blockhash), nil}, w)
			return
		}
		if rows.Next() {
			rows.Scan(&blockheight)
			rows.Close()
		} else {
			utils.JsonResponse(resp{1, fmt.Sprintf("no record in blockinfo where blockhash = %v", blockhash), nil}, w)
			return
		}
		log.Printf("Got blockheight: %v\n", blockhash)

	} else {
		utils.JsonResponse(resp{1, "Please give me blockid or blockhash parameter!", nil}, w)
		return
	}
	//blockheight got

	var txIDs []int64
	//根据blockid，到txinfo表拿txID
	rows, err = models.DB.Query("select txID from txinfo where blockID = ?", blockheight)
	if err != nil {
		utils.JsonResponse(resp{1, fmt.Sprintf("SQL error: select txID from txinfo where blockID = %v", blockheight), nil}, w)
		return
	}
	for rows.Next() {
		rows.Scan(&txID)
		txIDs = append(txIDs, txID)
	}
	rows.Close()
	log.Printf("txIDs: %v\n", txIDs)
	params := []interface{}{}
	for _, v := range txIDs {
		params = append(params, v)
	}
	rows, err = models.DB.Query("select txID,risktag from txhash where txID in "+utils.MakeQuestion(len(txIDs)), params...)
	if err != nil {
		utils.JsonResponse(resp{1, fmt.Sprintf("SQL error: select txID,risktag from txhash where txID in %v", txIDs), nil}, w)
		return
	}
	var txs []tx
	for rows.Next() {
		//TODO all rows.Close after Scan
		rows.Scan(&txID, &tag)
		txs = append(txs, tx{txID, tag})
	}
	rows.Close()

	utils.JsonResponse(resp{0, "OK", txs}, w)

}
