package models

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"log"
)

var (
	DB *sql.DB
)

func InitDB(dsn string) {
	var err error
	if len(dsn) == 0 {
		log.Fatal(`DSN is empty. `)
	}
	log.Printf("data source name: %v", dsn)
	DB, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Open mysql error: %v\n", err)
	}

	err = DB.Ping()
	if err != nil {
		log.Fatalf("Connect db error!")
	}

	DB.SetMaxIdleConns(20)
	DB.SetMaxOpenConns(20)

	//db, err := sql.Open("mysql", "root:@/kyt")
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//stmt, err := db.Prepare("insert into test set title=?,author=?")
	//if err != nil{
	//	log.Fatal(err)
	//}
	//stmt.Exec("title01", "eagle711")
	//

	//rows, _ := db.Query("select addrID,address,idtag from addresses limit 10")
	//for rows.Next() {
	//	var (
	//		id     int
	//		address  string
	//		idtag string
	//	)
	//	rows.Scan(&id,&address,&idtag)
	//	fmt.Printf("id:%v address:%v idtag:%v\n",id,address,idtag)
	//}
}
