package models

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"testing"
)

func TestSelect(t *testing.T) {
	//db, err := sql.Open("mysql", "root:@/kyt_BTC")
	db, err := sql.Open("mysql", "root:1qaz2wsx@tcp(aliyun.eagle711.win:7101)/kyt_BTC")
	if err != nil {
		log.Fatal(err)
	}
	//var (
	//	id     int
	//	title  string
	//	author string
	//)

	//rows, err := db.Query("select id,title,author from test where title = ?", 2)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//if rows.Next() {
	//	rows.Scan(&id, &title, &author)
	//}
	//t.Logf("id: %v title: %v author: %v\n", id, title, author)

	//v := []interface{}{2, 5}
	//rows, err := db.Query("select id,title,author from test where id in"+makeQuestion(len(v)), v...)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//for rows.Next() {
	//	rows.Scan(&id, &title, &author)
	//	t.Logf("id: %v title: %v author: %v\n", id, title, author)
	//}

	//rows, err := db.Query("select id,title,author from test")
	//if err != nil{
	//	log.Fatal(err)
	//}
	//for rows.Next(){
	//	rows.Scan(&id,&title,&author)
	//	t.Logf("id: %v title: %v author: %v\n", id, title, author)
	//}

	//aid := 100
	//rows, _ := db.Query("select addrID from addresses where address = ?", "1HSvEYwCFkDYDM3kGZgLbzSKiBdZ6QtvDC")
	//if rows.Next() {
	//	rows.Scan(&aid)
	//	t.Logf("aid: %v",aid)
	//}
	//
	rows3, err := db.Query("select a.addrID,a.risktag from addresses a join txin b on (a.addrID=b.addrID) where b.txID in (69,70);")
	if err != nil {
		log.Fatal(err)
	}
	var tag string
    addrID:=100
	for rows3.Next() {
		rows3.Scan( &addrID,&tag)
		t.Logf("tag: %v,addrID: %v", tag, addrID)
	}

	rows2, err := db.Query("select b.value,a.risktag from txhash a join (select txID,sum(value)as value from txout where addrID = 72 group by txID) b on (a.txID=b.txID)")
	if err != nil {
		log.Fatal(err)
	}
	var (
		risktag string
	)
	value := 100
	for rows2.Next() {
		rows2.Scan(&value, &risktag)
		t.Logf("risktag: %v,value: %v", risktag, value)
	}



}

func makeQuestion(n int) string {
	ret := "("
	for i := 0; i < n; i++ {
		ret += "?"
		if i < n-1 {
			ret += ","
		}
	}
	ret += ")"
	return ret
}
