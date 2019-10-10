package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/sunliang711/kyt/models"
	"github.com/sunliang711/kyt/utils"
)

type tx struct {
	Tag  string `json:"tag"`
	Hash string `json:"hash"`
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
	rows, err = models.DB.Query("select a.txID from txinfo a join txhash b on (a.txID=b.txID) where a.blockID = ? and b.risktag is not null", blockheight)
	if err != nil {
		utils.JsonResponse(resp{1, fmt.Sprintf("SQL error: select txID from txinfo where blockID = %v", blockheight), nil}, w)
		return
	}
	for rows.Next() {
		rows.Scan(&txID)
		txIDs = append(txIDs, txID)
	}
	rows.Close()
	var txs []tx
	log.Printf("txIDs: %v\n", txIDs)
	if len(txIDs) == 0 {
		w.Header().Set("Content-Disposition", "attachment; filename=txList.txt")
		w.Header().Set("Content-Type", req.Header.Get("Content-Type"))
		w.Header().Set("Content-Length", req.Header.Get("Content-Length"))

		json.NewEncoder(w).Encode(fmt.Sprintf("No suspicious transaction detected in block height: %d", blockheight))
		return
	}
	params := []interface{}{}
	for _, v := range txIDs {
		params = append(params, v)
	}
	rows, err = models.DB.Query("select txhash,risktag from txhash where txID in "+utils.MakeQuestion(len(txIDs)), params...)
	if err != nil {
		utils.JsonResponse(resp{1, fmt.Sprintf("SQL error: select txID,risktag from txhash where txID in %v", txIDs), nil}, w)
		return
	}
	var hash string
	for rows.Next() {
		rows.Scan(&hash, &tag)
		txs = append(txs, tx{tag, hash})
	}
	rows.Close()

	w.Header().Set("Content-Disposition", "attachment; filename=txList.txt")
	w.Header().Set("Content-Type", req.Header.Get("Content-Type"))
	w.Header().Set("Content-Length", req.Header.Get("Content-Length"))
	json.NewEncoder(w).Encode(&txs)
	// utils.JsonResponse(resp{0, "OK", txs}, w)

}

// blockTransactionGraph TODO
// 2019/09/24 15:56:39
// 找出一个块中所有交易，这些交易的srcs到dests的图
func blockTransactionGraph(w http.ResponseWriter, req *http.Request) {
	query := req.URL.Query()
	var (
		hash   string
		height int
		err    error
	)
	hash = query.Get("blockhash")
	heightStr := query.Get("blockid")
	if hash != "" {
		// height from table
		sql := "select blockID from blockinfo where blockhash = ?"
		rows, err := models.DB.Query(sql, hash)
		if err != nil {
			msg := fmt.Sprintf("Execute sql: %v error: %v", sql, err)
			log.Printf(msg)
			utils.JsonResponse(resp{1, msg, nil}, w)
			return
		}
		defer rows.Close()
		if rows.Next() {
			rows.Scan(&height)
		}
	} else if heightStr != "" {
		height, err = strconv.Atoi(heightStr)
		if err != nil {
			utils.JsonResponse(resp{1, "blockHeight invalid", nil}, w)
			return
		}
	} else {
		msg := fmt.Sprintf("no blockhash or blockheight!")
		log.Printf(msg)
		utils.JsonResponse(resp{1, msg, nil}, w)
		return
	}

	//get all txs by height
	sql := "select txID from txinfo where blockID = ?"
	rows, err := models.DB.Query(sql, height)
	if err != nil {
		msg := fmt.Sprintf("Execute sql: %v error: %v\n", sql, err)
		log.Printf(msg)
		utils.JsonResponse(resp{1, msg, nil}, w)
		return
	}
	// defer rows.Close()
	var txID int
	var txIDs []int
	for rows.Next() {
		rows.Scan(&txID)
		txIDs = append(txIDs, txID)
	}
	fmt.Printf("txID count in this block: %v\n", len(txIDs))

	// var (
	// 	allNodes []addressNode
	// 	allLinks []addressLink
	// )

	ret, err := TxsGraph(txIDs, -2)
	if err != nil {
		utils.JsonResponse(resp{1, "error", err}, w)
		return
	}
	utils.JsonResponse(resp{0, "OK", ret}, w)
	return

	// for _, id := range txIDs {
	// 	nodes, links, err := oneTxGraph(id)
	// 	if err != nil {
	// 		msg := fmt.Sprintf("oneTxGraph error: %v", err)
	// 		log.Printf(msg)
	// 		utils.JsonResponse(resp{1, msg, nil}, w)
	// 		return
	// 	}
	// 	allNodes = append(allNodes, nodes...)
	// 	allLinks = append(allLinks, links...)
	// }

	// ret := struct {
	// 	Nodes []addressNode `json:"nodes"`
	// 	Links []addressLink `json:"links"`
	// }{
	// 	allNodes,
	// 	allLinks,
	// }
	// utils.JsonResponse(resp{0, "OK", ret}, w)
}

// addressNode TODO
// 2019/09/24 19:03:57
type addressNode struct {
	ID          int    `json:"id"`          //addrID
	User        int    `json:"group"`       //userID
	Description string `json:"description"` //addrHash
}

// addressLink TODO
// 2019/09/24 19:06:54
type addressLink struct {
	// Source string `json:"source"`
	// Target string `json:"target"`
	Source int `json:"source"`
	Target int `json:"target"`
}

// makeRangeStr TODO
// 2019/09/25 15:58:39
func makeRangeStr(ids []int) string {
	ret := "("
	for i, id := range ids {
		ret += fmt.Sprintf("%d", id)
		if i != len(ids)-1 {
			ret += ","
		}
	}
	ret += ")"
	return ret
}

// txIDAddrID TODO
// 2019/09/25 16:26:54
type txIDAddrID struct {
	txID   int
	addrID int
}
type userIDAddress struct {
	userID  int
	address string
}

// txSrcDest TODO
// 2019/09/25 18:42:46
type txSrcDest struct {
	Src  []int
	Dest []int
}

type nodesLinks struct {
	Nodes []addressNode `json:"nodes"`
	Links []addressLink `json:"links"`
}

// TxsGraph TODO
// 2019/09/25 15:40:28
func TxsGraph(txIDs []int, paraAddrID int) (*nodesLinks, error) {
	fmt.Printf("len txIDs: %v\n", txIDs)
	if len(txIDs) == 0 {
		return &nodesLinks{}, nil
	}
	txIDs = utils.UniqIntSlice(txIDs)
	rangeStr := makeRangeStr(txIDs)
	// table txin
	sql := "select distinct txID,addrID from txin where txID in " + rangeStr
	rows, err := models.DB.Query(sql)
	if err != nil {
		log.Printf("Execute sql(%v) error: %v", sql, err)
		return nil, err
	}
	var (
		// makeRange
		addrIDs []int
		txIDMap = make(map[int]txSrcDest)

		// addrID int
		// tID     int

	)
	for rows.Next() {
		var tID int
		var addrID int

		rows.Scan(&tID, &addrID)
		addrIDs = append(addrIDs, addrID)
		v, exist := txIDMap[tID]
		if !exist {
			txIDMap[tID] = txSrcDest{
				Src:  []int{addrID},
				Dest: []int{},
			}
		} else {
			v.Src = append(v.Src, addrID)
			txIDMap[tID] = v
		}
	}
	// log.Printf("txIDMap: %v", txIDMap)

	rangeStr = makeRangeStr(addrIDs)
	// sql = fmt.Sprintf("select distinct c.addrID,c.userID,d.address from (select a.addrID,b.userID from  txin a join entities b on (a.addrID=b.addrID) where a.addrID in %s) c join addresses d on (c.addrID=d.addrID)", rangeStr)

	// sql = fmt.Sprintf("select c.addrID,c.address,d.userID from entities d right join (select a.addrID,a.address from addresses a where a.addrID in %s) c on (c.addrID=d.addrID)", rangeStr)
	sql = fmt.Sprintf("select a.addrID,a.address from addresses a where a.addrID in %s", rangeStr)
	rows, err = models.DB.Query(sql)
	if err != nil {
		log.Printf("Execute sql error: %v", err)
		return nil, err
	}
	var (
		srcNodes  []addressNode
		addrIDMap = make(map[int]*userIDAddress)
	)

	for rows.Next() {
		var addrID int
		var address string

		rows.Scan(&addrID, &address)
		// fmt.Printf("addrID: %v\n", addrID)
		addrIDMap[addrID] = &userIDAddress{0, address}
	}
	// log.Printf("addrIDMap: %v", addrIDMap)
	sql = fmt.Sprintf("select addrID,userID from entities where addrID in %s", rangeStr)
	rows, err = models.DB.Query(sql)
	if err != nil {
		log.Printf("Execute sql error: %v", err)
		return nil, err
	}
	for rows.Next() {
		var addrID int
		var userID int
		rows.Scan(&addrID, &userID)
		addrIDMap[addrID].userID = userID
	}

	uniqueNode := make(map[int]bool)

	for _, sd := range txIDMap {
		for _, src := range sd.Src {
			addrID := src
			// fmt.Printf("addrIDMap[%v]:%v", addrID, addrIDMap[addrID])
			_, exist := uniqueNode[addrID]
			// fmt.Printf("src addrID: %v\n", addrID)
			if !exist {
				uniqueNode[addrID] = true

				// if event {
				// 	// fmt.Printf("EVENT..........\n")
				// 	// sql = "select count(*) from eventrecord where addrID = ?"
				// 	// rows, _ := models.DB.Query(sql, addrID)
				// 	// var count int
				// 	// if rows.Next() {
				// 	// 	rows.Scan(&count)
				// 	// 	fmt.Printf("addrID: %v count: %v\n", addrID, count)
				// 	// }
				// 	// rows.Close()
				// 	// if count > 0 {
				// 	fmt.Printf("add addrID: %v\n", addrID)
				// 	srcNodes = append(srcNodes, addressNode{addrID, addrIDMap[addrID].userID, addrIDMap[addrID].address})
				// 	// }

				// } else {
				// 	srcNodes = append(srcNodes, addressNode{addrID, addrIDMap[addrID].userID, addrIDMap[addrID].address})
				// }
				srcNodes = append(srcNodes, addressNode{addrID, addrIDMap[addrID].userID, addrIDMap[addrID].address})
				//
			}
		}
		// srcNodes = append(srcNodes, addressNode{v[0].addrID, addrIDMap[v[0].addrID].userID, addrIDMap[v[0].addrID].address})
	}

	// --------------------------------------------------
	// -- Table txout
	// --------------------------------------------------
	rangeStr = makeRangeStr(txIDs)
	sql = "select distinct txID,addrID from txout where txID in " + rangeStr
	rows, err = models.DB.Query(sql)
	if err != nil {
		log.Printf("Execute sql error: %v", err)
		return nil, err
	}

	addrIDs = []int{}
	for rows.Next() {
		var tID int
		var addrID int
		rows.Scan(&tID, &addrID)
		addrIDs = append(addrIDs, addrID)
		// txIDAddrIDs = append(txIDAddrIDs,txIDAddrID{tID,addrID})
		// txIDMap[tID] = append(txIDMap[tID], txIDAddrID{tID, addrID})
		v, exist := txIDMap[tID]
		if !exist {
			txIDMap[tID] = txSrcDest{
				Src:  []int{},
				Dest: []int{addrID},
			}
		} else {
			v.Dest = append(v.Dest, addrID)
			txIDMap[tID] = v
		}
	}
	// log.Printf("txIDMap: %v", txIDMap)

	rangeStr = makeRangeStr(addrIDs)
	// sql = fmt.Sprintf("select distinct c.addrID,c.userID,d.address from (select a.addrID,b.userID from  txin a join entities b on (a.addrID=b.addrID) where a.addrID in %s) c join addresses d on (c.addrID=d.addrID)", rangeStr)

	// sql = fmt.Sprintf("select c.addrID,c.address,d.userID from entities d left join (select a.addrID,a.address from addresses a where a.addrID in %s) c on (c.addrID=d.addrID)", rangeStr)
	sql = fmt.Sprintf("select addrID,address from addresses where addrID in %s", rangeStr)
	rows, err = models.DB.Query(sql)
	if err != nil {
		log.Printf("Execute sql error: %v", err)
		return nil, err
	}
	var (
		destNodes []addressNode
		// destAddrIDMap = make(map[int]*userIDAddress)
	)

	for rows.Next() {
		var addrID int
		var address string
		rows.Scan(&addrID, &address)
		// fmt.Printf("addrID: %v\n", addrID)
		addrIDMap[addrID] = &userIDAddress{0, address}
	}
	// log.Printf("addrIDMap: %v", addrIDMap)
	sql = fmt.Sprintf("select addrID,userID from entities where addrID in %s", rangeStr)
	rows, err = models.DB.Query(sql)
	if err != nil {
		log.Printf("Execute sql error: %v", err)
		return nil, err
	}
	for rows.Next() {
		var addrID int
		var userID int
		rows.Scan(&addrID, &userID)
		addrIDMap[addrID].userID = userID
	}

	for _, sd := range txIDMap {
		for _, dest := range sd.Dest {
			addrID := dest
			// _ = addrID
			// fmt.Printf("addrIDMap[%v]:%v", addrID, destAddrIDMap[addrID])
			// fmt.Printf("addrID: %v\n", addrID)
			if addrID == -1 {
				continue
			}
			// if addrIDMap[addrID] == nil {
			// 	continue
			// }
			_, exist := uniqueNode[addrID]
			if !exist {
				uniqueNode[addrID] = true
				destNodes = append(destNodes, addressNode{addrID, addrIDMap[addrID].userID, addrIDMap[addrID].address})
			}
		}
		// srcNodes = append(srcNodes, addressNode{v[0].addrID, addrIDMap[v[0].addrID].userID, addrIDMap[v[0].addrID].address})
	}
	// fmt.Printf("srcNodes: %v", srcNodes)
	// fmt.Printf("destNodes: %v", destNodes)

	//make link
	var links []addressLink
	uniqueLinks := make(map[string]bool)
	usefulNodes := make(map[int]bool)
	for _, v := range txIDMap {
		for _, s := range v.Src {
			for _, d := range v.Dest {
				// log.Printf("s:%v", s)
				// log.Printf("d:%v", d)
				if s == -1 || d == -1 {
					continue
				}
				if s == d {
					continue
				}

				if paraAddrID != -2 && paraAddrID != s && paraAddrID != d {
					continue
				}

				_, exist := uniqueLinks[fmt.Sprintf("%v%v", s, d)]
				if !exist {
					links = append(links, addressLink{s, d})
					uniqueLinks[fmt.Sprintf("%v%v", s, d)] = true
					usefulNodes[s] = true
					usefulNodes[d] = true
				}
				// if addrIDMap[s] == nil || addrIDMap[d] == nil {
				// 	continue
				// }
				// links = append(links, addressLink{fmt.Sprintf("%d", s), fmt.Sprintf("%d", d)})
			}
		}

	}
	var retSrcNodes []addressNode
	var retDestNodes []addressNode
	for _, n := range srcNodes {
		if _, exist := usefulNodes[n.ID]; exist {
			retSrcNodes = append(retSrcNodes, n)
		}
	}
	for _, n := range destNodes {
		if _, exist := usefulNodes[n.ID]; exist {
			retDestNodes = append(retDestNodes, n)
		}
	}

	ret := &nodesLinks{
		append(retSrcNodes, retDestNodes...),
		links,
	}
	return ret, nil
}

// oneTxGraph TODO
// 2019/09/24 19:01:38
func oneTxGraph(txID int) (addressNodes []addressNode, addressLinks []addressLink, err error) {
	// txin table
	// sql1 := "select a.addrID,b.address from txin a join addresses b on (a.addrID=b.addrID) where a.txID = ?"
	// sql2 := "select a.addrID,b.userID from txin a join entities b on (a.addrID=b.addrID) where a.txID = ?"
	// sql1 + sql2 => sql

	//tx sources
	sql := "select c.addrID,c.address,d.userID from entities d join (select a.addrID,b.address from txin a join addresses b on (a.addrID=b.addrID) where a.txID = ?) c on (d.addrID=c.addrID)"
	log.Printf("execute sql: %v", sql)
	rows, err := models.DB.Query(sql, txID)
	if err != nil {
		log.Printf("execute sql error: %v", err)
		return nil, nil, err
	}
	var srcNodes []addressNode
	for rows.Next() {
		var (
			addrID  int
			address string
			userID  int
		)
		rows.Scan(&addrID, &address, &userID)
		srcNodes = append(srcNodes, addressNode{addrID, userID, address})
	}

	// txout table
	// tx dests
	sql = "select c.addrID,c.address,d.userID from entities d join (select a.addrID,b.address from txout a join addresses b on (a.addrID=b.addrID) where a.txID = ?) c on (d.addrID=c.addrID)"
	rows, err = models.DB.Query(sql, txID)
	if err != nil {
		log.Printf("execute sql: %v error: %v", sql, err)
		return nil, nil, err
	}

	var destNodes []addressNode
	for rows.Next() {
		var (
			addrID  int
			address string
			userID  int
		)
		rows.Scan(&addrID, &address, &userID)
		destNodes = append(destNodes, addressNode{addrID, userID, address})
	}

	addressNodes = append(addressNodes, srcNodes...)
	addressNodes = append(addressNodes, destNodes...)

	//link = srcs => dests

	for _, src := range srcNodes {
		for _, dest := range destNodes {
			addressLinks = append(addressLinks, addressLink{
				Source: src.ID,
				Target: dest.ID,
			})
		}
	}
	return
}

func blockHeightByHash(w http.ResponseWriter, req *http.Request) {
	query := req.URL.Query()
	hash := query.Get("hash")
	if hash == "" {
		utils.JsonResponse(resp{1, "need query parameter hash", nil}, w)
		return
	}

	sql := "select blockID from blockinfo where blockhash = ?"
	rows, err := models.DB.Query(sql, hash)
	if err != nil {
		utils.JsonResponse(resp{1, "query block height error", err}, w)
		return
	}
	defer rows.Close()
	var height int
	if rows.Next() {
		rows.Scan(&height)
	}
	utils.JsonResponse(resp{0, "OK", height}, w)

}
func blockHashByHeight(w http.ResponseWriter, req *http.Request) {
	query := req.URL.Query()
	height := query.Get("height")
	if height == "" {
		utils.JsonResponse(resp{1, "need query parameter height", nil}, w)
		return
	}
	sql := "select blockhash from blockinfo where blockID = ?"
	rows, err := models.DB.Query(sql, height)
	if err != nil {
		utils.JsonResponse(resp{1, "query block hash error", err}, w)
		return
	}
	defer rows.Close()
	var hash string
	if rows.Next() {
		rows.Scan(&hash)
	}
	utils.JsonResponse(resp{0, "OK", hash}, w)
}

func blockTags(w http.ResponseWriter, req *http.Request) {
	query := req.URL.Query()
	var (
		hash   string
		height int
		err    error
	)
	hash = query.Get("blockhash")
	heightStr := query.Get("blockid")
	if hash != "" {
		// height from table
		sql := "select blockID from blockinfo where blockhash = ?"
		rows, err := models.DB.Query(sql, hash)
		if err != nil {
			msg := fmt.Sprintf("Execute sql: %v error: %v", sql, err)
			log.Printf(msg)
			utils.JsonResponse(resp{1, msg, nil}, w)
			return
		}
		defer rows.Close()
		if rows.Next() {
			rows.Scan(&height)
		}
	} else if heightStr != "" {
		height, err = strconv.Atoi(heightStr)
		if err != nil {
			utils.JsonResponse(resp{1, "blockHeight invalid", nil}, w)
			return
		}
	} else {
		msg := fmt.Sprintf("no blockhash or blockheight!")
		log.Printf(msg)
		utils.JsonResponse(resp{1, msg, nil}, w)
		return
	}

	sql := "select distinct a.risktag from txhash a join txinfo b on (a.txID=b.txID) where b.blockID=?"
	rows, err := models.DB.Query(sql, height)
	if err != nil {
		utils.JsonResponse(resp{1, "query risktag error", err}, w)
		return
	}
	var (
		tag  string
		tags []string
	)
	for rows.Next() {
		rows.Scan(&tag)
		if tag != "" {
			tags = append(tags, tag)
		}
	}
	utils.JsonResponse(resp{0, "OK", tags}, w)
}
