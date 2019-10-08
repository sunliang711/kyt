package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/sunliang711/kyt/models"
	"github.com/sunliang711/kyt/utils"
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
	rows, err = models.DB.Query("select distinct a.idtag from addresses a join multiple b on (a.addrID = b.addrID) where b.txID = ?", txID)
	if err != nil {
		utils.JsonResponse(resp{1, "internal db error", nil}, w)
		return
	}
	for rows.Next() {
		var idtag string
		rows.Scan(&idtag)
		if idtag == "" {
			continue
		}
		idtag = strings.ReplaceAll(idtag, "'", "\"")
		fmt.Printf("multiple idtag: %v\n", idtag)
		var tagArray [][]string
		err = json.Unmarshal([]byte(idtag), &tagArray)
		if err != nil {
			continue
		}
		fmt.Printf("tagArray: %v\n", tagArray)
		sender = append(sender, tagArray[0][0])
	}
	rows.Close()

	//IF no data,then issue the following:
	//select addrID from txin where txID = TXID;
	rows, err = models.DB.Query("select distinct a.idtag from addresses a join txin b on (a.addrID = b.addrID) where b.txID = ?", txID)
	if err != nil {
		utils.JsonResponse(resp{1, "internal db error", nil}, w)
		return
	}
	for rows.Next() {
		var idtag string
		rows.Scan(&idtag)
		if idtag == "" {
			continue
		}
		idtag = strings.ReplaceAll(idtag, "'", "\"")
		fmt.Printf("txin idtag: %v\n", idtag)
		var tagArray [][]string
		err = json.Unmarshal([]byte(idtag), &tagArray)
		if err != nil {
			continue
		}

		fmt.Printf("tagArray: %v\n", tagArray)
		sender = append(sender, tagArray[0][0])
	}
	rows.Close()

	//receiver
	//select addrID from txout where txID = TXID;
	rows, err = models.DB.Query("select distinct a.idtag from addresses a join txout b on (a.addrID = b.addrID) where b.txID = ?", txID)
	if err != nil {
		utils.JsonResponse(resp{1, "internal db error", nil}, w)
		return
	}
	for rows.Next() {
		var idtag string
		rows.Scan(&idtag)
		if idtag == "" {
			continue
		}
		idtag = strings.ReplaceAll(idtag, "'", "\"")
		fmt.Printf("txout idtag: %v\n", idtag)
		var tagArray [][]string
		err = json.Unmarshal([]byte(idtag), &tagArray)
		if err != nil {
			continue
		}
		fmt.Printf("tagArray: %v\n", tagArray)
		receiver = append(receiver, tagArray[0][0])
	}
	rows.Close()

	sender = utils.UniqStringSlice(sender)
	receiver = utils.UniqStringSlice(receiver)
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
			log.Printf("eventtag: %v", eventtag)

			w.Header().Set("Content-Disposition", "attachment; filename=txList.txt")
			w.Header().Set("Content-Type", req.Header.Get("Content-Type"))
			w.Header().Set("Content-Length", req.Header.Get("Content-Length"))
			json.NewEncoder(w).Encode(&hashTags)

			// utils.JsonResponse(resp{0, "OK", hashTags}, w)
			return
		} else {
			log.Printf("eventtag is empty")
			idChan := make(chan int, 4096)
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
			if len(ids) == 0 {
				utils.JsonResponse(resp{1, fmt.Sprintf("no data  with txhash: %v", txhash), nil}, w)
				return
			}
			sql = "select txhash,eventtag from txhash where txID in " + utils.MakeQuestion(len(ids))
			rows, err = models.DB.Query(sql, ids...)
			var hashtags []*hashTag
			for rows.Next() {
				rows.Scan(&hash, eventtag)
				log.Printf("get hash:%v,eventtag:%v.", hash, eventtag)
				hashtags = append(hashtags, &hashTag{hash, eventtag})
			}
			rows.Close()
			log.Printf("eventtag is empty")
			w.Header().Set("Content-Disposition", "attachment; filename=txList.txt")
			w.Header().Set("Content-Type", req.Header.Get("Content-Type"))
			w.Header().Set("Content-Length", req.Header.Get("Content-Length"))
			json.NewEncoder(w).Encode(&hashtags)
			// utils.JsonResponse(resp{0, "ok", hashtags}, w)
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
	max--
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
	max--
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
func transactionTag(w http.ResponseWriter, req *http.Request) {
	query := req.URL.Query()
	var ()
	txhash := query.Get("txhash")
	if len(txhash) == 0 {
		utils.JsonResponse(resp{1, "need parameter: txhash", nil}, w)
		return
	}

	sql := "select risktag from txhash where txhash = ?"
	rows, err := models.DB.Query(sql, txhash)
	if err != nil {
		utils.JsonResponse(resp{1, "internal db error", nil}, w)
		log.Printf("SQL error: %v", sql)
		return
	}
	defer rows.Close()
	var tag string
	if rows.Next() {
		rows.Scan(&tag)
	}
	utils.JsonResponse(resp{0, "OK", tag}, w)
}

func transactionGraph(w http.ResponseWriter, req *http.Request) {
	log.Printf("Enter transactionGraph")
	query := req.URL.Query()
	txhash := query.Get("txhash")
	if len(txhash) == 0 {
		utils.JsonResponse(resp{1, "need parameter: txhash", nil}, w)
		return
	}

	sql := "select txID,txhash,eventtag from txhash where txhash = ?"
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
		rows.Scan(&txID, &hash, &eventtag)
		rows.Close()
	}
	log.Printf("txID: %v ,hash: %v ,eventTag: %v\n", txID, hash, eventtag)
	idChan := make(chan *idGroup, 1024)
	linkChan := make(chan *link, 1024)
	var (
		idGroups []*idGroup
		links    []*link
	)
	idChanDone := make(chan struct{})
	linkChanDone := make(chan struct{})

	uniqueID := make(map[string]bool)
	go func() {
		for id := range idChan {
			_, exist := uniqueID[id.ID]
			if !exist {
				uniqueID[id.ID] = true
				idGroups = append(idGroups, id)
			}
		}
		close(idChanDone)
	}()
	go func() {
		for l := range linkChan {
			links = append(links, l)
		}
		close(linkChanDone)
	}()
	if eventtag == "" {
		//前后追溯3层
		first := true
		err = findPrevs2(txID, hash, idChan, linkChan, 3, first)
		if err != nil {
			utils.JsonResponse(resp{1, "find prevs error", nil}, w)
			log.Printf("find prevs error: %v", err)
			return
		}
		err = findNexts2(txID, hash, idChan, linkChan, 3)
		if err != nil {
			utils.JsonResponse(resp{1, "find nexts error", nil}, w)
			log.Printf("find nexts error: %v", err)
			return
		}

	} else {
		//返回该事件所有
		//使用eventtag查询回溯路径图
		first := true
		err = findPrevs3(txID, hash, idChan, linkChan, eventtag, first)
		if err != nil {
			utils.JsonResponse(resp{1, "find preves with tag error", nil}, w)
			return
		}
		err = findNexts3(txID, hash, idChan, linkChan, eventtag)
		if err != nil {
			utils.JsonResponse(resp{1, "find nexts error", nil}, w)
			return
		}
	}
	close(idChan)
	close(linkChan)

	<-idChanDone
	<-linkChanDone
	res := struct {
		Nodes []*idGroup `json:"nodes"`
		Links []*link    `json:"links"`
	}{
		idGroups,
		links,
	}
	utils.JsonResponse(resp{0, "OK", res}, w)

}
func findPrev2(txID int) ([]int, []string, error) {
	sql := "select a.prev_txID,b.txhash from txin a join txhash b on (a.prev_txID=b.txID) where a.txID = ?"
	rows, err := models.DB.Query(sql, txID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var (
		id  int
		ids []int

		hash   string
		hashes []string
	)
	for rows.Next() {
		rows.Scan(&id, &hash)
		log.Printf("get id:%v hash: %v", id, hash)
		ids = append(ids, id)
		hashes = append(hashes, hash)
	}
	return ids, hashes, nil
}

type idGroup struct {
	ID    string `json:"id"`
	Group int    `json:"group"`
}
type link struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Value  int    `json:"value"`
}

func txID2Hash(txID int) string {
	sql := "select txhash from txhash where txID = ?"
	rows, err := models.DB.Query(sql, txID)
	if err != nil {
		return "sql execute error"
	}
	defer rows.Close()
	var hash string
	if rows.Next() {
		rows.Scan(&hash)
	}
	log.Printf("txID2Hash: %v->%v", txID, hash)
	return hash
}
func findPrevs2(txID int, hash string, idChan chan *idGroup, linkChan chan *link, max uint, first bool) error {
	if max == 0 {
		return nil
	}
	group := 2
	if first {
		group = 1
		first = !first
	}
	if max == 1 {
		//the last layer
		group = 3
	}
	max--
	idChan <- &idGroup{
		fmt.Sprintf("%v", hash),
		group}
	prevIDs, prevHashes, err := findPrev2(txID)
	if err != nil {
		return err
	}
	for index, pid := range prevIDs {
		if pid != -1 {
			idChan <- &idGroup{
				fmt.Sprintf("%v", prevHashes[index]),
				group}
			linkChan <- &link{
				fmt.Sprintf("%v", prevHashes[index]),
				fmt.Sprintf("%v", hash),
				1}
		}
	}

	fmt.Println("recursive find")
	for index, pid := range prevIDs {
		if pid != -1 {
			err := findPrevs2(pid, prevHashes[index], idChan, linkChan, max, first)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func findNext2(txID int) ([]int, []string, error) {
	sql := "select a.txID,b.txhash from txin a join txhash b on(a.txID=b.txID) where a.prev_txID = ?"
	rows, err := models.DB.Query(sql, txID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var (
		id  int
		ids []int

		hash   string
		hashes []string
	)
	for rows.Next() {
		rows.Scan(&id, &hash)
		log.Printf("get id:%v hash: %v", id, hash)
		ids = append(ids, id)
		hashes = append(hashes, hash)
	}
	return ids, hashes, nil
}
func findNexts2(txID int, hash string, idChan chan *idGroup, linkChan chan *link, max uint) error {
	if max == 0 {
		return nil
	}
	group := 2
	if max == 1 {
		//the last layer
		group = 3
	}
	max--
	nextIDs, nextHashes, err := findNext2(txID)
	if err != nil {
		return err
	}
	for index, pid := range nextIDs {
		if pid != -1 {
			idChan <- &idGroup{
				fmt.Sprintf("%v", nextHashes[index]),
				group}
			linkChan <- &link{
				fmt.Sprintf("%v", nextHashes[index]),
				fmt.Sprintf("%v", hash),
				1}
		}
	}
	fmt.Println("recursive find")
	for index, pid := range nextIDs {
		if pid != -1 {
			// go func(pid int) error {
			err = findNexts2(pid, nextHashes[index], idChan, linkChan, max)
			if err != nil {
				return err
			}
			// 	return nil
			// }(pid)
		}
	}

	return nil
}

func findPrev3(txID int, tag string) ([]int, []string, error) {
	// sql := "select prev_txID from txin where txID = ?"
	sql := "select a.prev_txID,b.txhash from txin a join txhash b on (a.txID=b.txID) where a.txID = ? and b.risktag is not null"
	rows, err := models.DB.Query(sql, txID)
	if err != nil {
		log.Printf("findPrev3 error: %v", err)
		return nil, nil, err
	}
	defer rows.Close()
	var (
		id     int
		hash   string
		ids    []int
		hashes []string
	)
	for rows.Next() {
		rows.Scan(&id, &hash)
		log.Printf("get id: %v", id)
		log.Printf("get hash: %v", hash)
		ids = append(ids, id)
		hashes = append(hashes, hash)
	}
	return ids, hashes, nil
}

func findPrevs3(txID int, hash string, idChan chan *idGroup, linkChan chan *link, tag string, first bool) error {
	group := 1
	if !first {
		group = 2
	}
	if first {
		first = !first
	}
	idChan <- &idGroup{
		fmt.Sprintf("%v", hash),
		group}
	prevIDs, prevHashes, err := findPrev3(txID, tag)
	if err != nil {
		return err
	}
	for index, pid := range prevIDs {
		if pid != -1 {
			idChan <- &idGroup{
				fmt.Sprintf("%v", prevHashes[index]),
				group}
			linkChan <- &link{
				fmt.Sprintf("%v", prevHashes[index]),
				fmt.Sprintf("%v", hash),
				1}
		}
	}
	fmt.Println("recursive find")
	for index, pid := range prevIDs {
		if pid != -1 {
			err = findPrevs3(pid, prevHashes[index], idChan, linkChan, tag, first)
			if err != nil {
				return err
			}

		}
	}

	return nil
}

func findNext3(txID int, tag string) ([]int, []string, error) {
	// sql := "select txID from txin where prev_txID = ?"
	sql := "select a.txID,b.txhash from txin a join txhash b on (a.txID=b.txID) where a.prev_txID = ? and b.risktag is not null"
	rows, err := models.DB.Query(sql, txID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var (
		id  int
		ids []int

		hash   string
		hashes []string
	)
	for rows.Next() {
		rows.Scan(&id, &hash)
		log.Printf("get id:%v hash: %v", id, hash)
		ids = append(ids, id)
		hashes = append(hashes, hash)
	}
	return ids, hashes, nil
}
func findNexts3(txID int, hash string, idChan chan *idGroup, linkChan chan *link, tag string) error {
	nextIDs, nextHashes, err := findNext3(txID, tag)
	if err != nil {
		return err
	}
	for index, pid := range nextIDs {
		if pid != -1 {
			idChan <- &idGroup{
				fmt.Sprintf("%v", nextHashes[index]),
				1}
			linkChan <- &link{
				fmt.Sprintf("%v", nextHashes[index]),
				fmt.Sprintf("%v", hash),
				1}
		}
	}
	for index, pid := range nextIDs {
		if pid != -1 {
			err = findNexts3(pid, nextHashes[index], idChan, linkChan, tag)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
