package handlers

import (
	"database/sql"
	"fmt"
	"github.com/sunliang711/kyt/models"
	"github.com/sunliang711/kyt/utils"
	"log"
	"net/http"
)

type txSenderReceiver struct {
	Sender   []string `json:"Sender"`
	Receiver []string `json:"Receiver"`
}

func transactionIdentity(w http.ResponseWriter, req *http.Request) {
	query := req.URL.Query()
	var (
		err      error
		rows     *sql.Rows
		txID     int64
		address  string
		sender   []string
		receiver []string
	)
	txhash := query.Get("txhash")
	if len(txhash) == 0 {
		utils.JsonResponse(resp{1, "need parameter: txhash", nil}, w)
		return
	}

	//get txID by txhash
	rows, err = models.DB.Query("select txID from txhash where txhash = ?", txhash)
	if err != nil {
		utils.JsonResponse(resp{1, "internal db error", nil}, w)
		return
	}
	if rows.Next() {
		rows.Scan(&txID)
		rows.Close()
	} else {
		utils.JsonResponse(resp{1, "no such txID in table txhash", nil}, w)
		return
	}

	//select addrID from multiple where txID = TXID;
	//rows, err = db.Query("select addrID from multiple where txID = ?", txID)
	rows, err = models.DB.Query("select a.address from addresses a join multiple b on (a.addrID = b.addrID) where b.txID = ?", txID)
	if err != nil {
		utils.JsonResponse(resp{1, "internal db error", nil}, w)
		return
	}
	for rows.Next() {
		rows.Scan(&address)
		sender = append(sender, address)
	}
	rows.Close()

	//IF no data,then issue the following:
	//select addrID from txin where txID = TXID;
	rows, err = models.DB.Query("select a.address from addresses a join txin b on (a.addrID = b.addrID) where b.txID = ?", txID)
	if err != nil {
		utils.JsonResponse(resp{1, "internal db error", nil}, w)
		return
	}
	for rows.Next() {
		rows.Scan(&address)
		sender = append(sender, address)
	}
	rows.Close()

	//receiver
	//select addrID from txout where txID = TXID;
	rows, err = models.DB.Query("select a.address from addresses a join txout b on (a.addrID = b.addrID) where b.txID = ?", txID)
	if err != nil {
		utils.JsonResponse(resp{1, "internal db error", nil}, w)
		return
	}
	for rows.Next() {
		rows.Scan(&address)
		receiver = append(receiver, address)
	}
	rows.Close()

	sender = utils.UniqStringSlice(sender)
	txSR := txSenderReceiver{sender, receiver}

	utils.JsonResponse(resp{0, "OK", &txSR}, w)

}

//type hashTag struct {
//	Hash string `json:"hash"`
//	Tag  string `json:"tag"`
//}

func transactionTxList(w http.ResponseWriter, req *http.Request) {
	query := req.URL.Query()
	var ()
	txhash := query.Get("txhash")
	if len(txhash) == 0 {
		utils.JsonResponse(resp{1, "need parameter: txhash", nil}, w)
		return
	}

	//select txID,eventtag from txhash where txhash = ?
	//if has data:
	//				select txhash from txhash where eventtag = ?
	//if no data:
	//				forward,backward n layer
	sql := "select txID,eventtag from txhash where txhash = ?"
	rows, err := models.DB.Query(sql, txhash)
	if err != nil {
		utils.JsonResponse(resp{1, "internal db error", nil}, w)
		log.Printf("SQL error: %v", sql)
		return
	}
	var (
		txID     int
		eventtag string
		hash     string
	)
	if rows.Next() {
		rows.Scan(&txID, &eventtag)
		rows.Close()
		log.Printf("get txID: %v", txID)
		if len(eventtag) > 0 {
			log.Printf("eventtag: %v", eventtag)
			sql = "select txhash,eventtag from txhash where eventtag = ?"
			rows, err = models.DB.Query(sql, eventtag)
			if err != nil {
				utils.JsonResponse(resp{1, "internal db error", nil}, w)
				log.Printf("SQL error: %v", sql)
				return
			}

			var hashTags []*hashTag
			for rows.Next() {
				rows.Scan(&hash, &eventtag)
				log.Printf("hash: %v,eventtag: %v", hash, eventtag)
				hashTags = append(hashTags, &hashTag{hash, eventtag})
			}
			rows.Close()
			utils.JsonResponse(resp{0, "OK", hashTags}, w)
			return
		} else {
			log.Printf("eventtag is empty")
			idChan := make(chan int, 1000)
			var ids []interface{}
			go func() {
				for id := range idChan {
					ids = append(ids, id)
				}
			}()
			err = findPrevs(txID, idChan, 5)
			if err != nil {
				utils.JsonResponse(resp{1, "find prevs error", nil}, w)
				return
			}
			err = findNexts(txID, idChan, 3)
			if err != nil {
				utils.JsonResponse(resp{1, "find nexts error", nil}, w)
				return
			}
			close(idChan)
			log.Printf("ids: %v", ids)
			sql = "select txhash,eventtag from txhash where txID in " + utils.MakeQuestion(len(ids))
			rows, err = models.DB.Query(sql, ids...)
			var hashtags []*hashTag
			for rows.Next() {
				rows.Scan(&hash, eventtag)
				log.Printf("get hash:%v,eventtag:%v.", hash, eventtag)
				hashtags = append(hashtags, &hashTag{hash, eventtag})
			}
			rows.Close()
			utils.JsonResponse(resp{0, "ok", hashtags}, w)
			return
		}
	} else {
		utils.JsonResponse(resp{1, fmt.Sprintf("no such tx with txhash: %v", txhash), nil}, w)
		return
	}
}

func findPrev(txID int) ([]int, error) {
	sql := "select prev_txID from txin where txID = ?"
	rows, err := models.DB.Query(sql, txID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var (
		id  int
		ids []int
	)
	for rows.Next() {
		rows.Scan(&id)
		log.Printf("get id:%v", id)
		ids = append(ids, id)
	}
	return ids, nil
}
func findPrevs(txID int, idChan chan int, max uint) error {
	if max == 0 {
		return nil
	}
	max -= 1
	prevs, err := findPrev(txID)
	if err != nil {
		return err
	}
	for _, pid := range prevs {
		if pid != -1 {
			idChan <- pid
		}
	}

	for _, pid := range prevs {
		if pid != -1 {
			err = findPrevs(pid, idChan, max)
			if err != nil {
				return err
			}

		}
	}

	return nil
}

func findNext(txID int) ([]int, error) {
	sql := "select txID from txin where prev_txID = ?"
	rows, err := models.DB.Query(sql, txID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var (
		id  int
		ids []int
	)
	for rows.Next() {
		rows.Scan(&id)
		log.Printf("get id:%v", id)
		ids = append(ids, id)
	}
	return ids, nil
}
func findNexts(txID int, idChan chan int, max uint) error {
	if max == 0 {
		return nil
	}
	max -= 1
	nexts, err := findNext(txID)
	if err != nil {
		return err
	}
	for _, pid := range nexts {
		if pid != -1 {
			idChan <- pid
		}
	}
	for _, pid := range nexts {
		if pid != -1 {
			err = findNexts(pid, idChan, max)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func transactionGraph(w http.ResponseWriter, req *http.Request) {
	query := req.URL.Query()
	var ()
	txhash := query.Get("txhash")
	if len(txhash) == 0 {
		utils.JsonResponse(resp{1, "need parameter: txhash", nil}, w)
		return
	}



}
