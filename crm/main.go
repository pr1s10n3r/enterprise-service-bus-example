package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

var database *sql.DB

type Client struct {
	ID        int64     `json:"id"`
	FullName  string    `json:"fullname"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func getParamOr(ctx *gin.Context, key, alt string) string {
	value := ctx.Query(key)
	if value == "" {
		return alt
	}
	return value
}

func createClient(ctx *gin.Context) {
	client := new(Client)

	if err := ctx.BindJSON(client); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}

	client.CreatedAt = time.Now()
	client.UpdatedAt = client.CreatedAt

	stmt, _ := database.Prepare("INSERT INTO clients (fullname, email, phone) VALUES (?, ?, ?)")
	defer stmt.Close()

	result, err := stmt.Exec(client.FullName, client.Email, client.Phone)
	if err != nil {
		log.Printf("unable to insert client: %s\n", err)
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	clientID, _ := result.LastInsertId()
	client.ID = clientID

	ctx.JSON(http.StatusCreated, client)
}

func getClients(ctx *gin.Context) {
	offset, err := strconv.ParseUint(getParamOr(ctx, "offset", "0"), 10, 64)
	if err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}

	limit, err := strconv.ParseUint(getParamOr(ctx, "limit", "50"), 10, 64)
	if err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}

	stmt, err := database.Prepare("SELECT id, fullname, email, phone, created_at, updated_at FROM clients LIMIT ?, ?")
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer stmt.Close()

	rows, err := stmt.Query(offset, limit)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	clients := make([]Client, 0)
	for rows.Next() {
		client := Client{}

		err := rows.Scan(&client.ID, &client.FullName, &client.Email, &client.Phone, &client.CreatedAt, &client.UpdatedAt)
		if err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		clients = append(clients, client)
	}

	ctx.JSON(http.StatusOK, clients)
}

func main() {
	if err := godotenv.Load(); err != nil {
		panic(err)
	}

	initDatabase()

	router := gin.Default()

	client := router.Group("/client")
	client.POST("/", createClient)
	client.GET("/", getClients)

	router.Run(":3000")
}

func initDatabase() {
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s)/%s?parseTime=true",
		os.Getenv("MYSQL_USERNAME"),
		os.Getenv("MYSQL_PASSWORD"),
		os.Getenv("MYSQL_HOST"),
		os.Getenv("MYSQL_DATABASE"),
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}

	if err = db.Ping(); err != nil {
		panic(err)
	}

	database = db
}
