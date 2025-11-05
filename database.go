package main

import (
	"database/sql"
	"log"
	"time"

	_ "modernc.org/sqlite"
)

var database *sql.DB

var seedUsers = []User{
	{
		Id:       1,
		Username: "zagreus",
		Password: "cerberus",
	},
	{
		Id:       2,
		Username: "melinoe",
		Password: "b4d3ec1",
	},
}

var seedProducts = []Product{
	{Id: 1, Name: "Americano", Price: 100, Description: "Espresso, diluted for a lighter experience"},
	{Id: 2, Name: "Cappuccino", Price: 110, Description: "Espresso with steamed milk"},
	{Id: 3, Name: "Espresso", Price: 90, Description: "A strong shot of coffee"},
	{Id: 4, Name: "Macchiato", Price: 120, Description: "Espresso with a small amount of milk"},
}

func initDB() {
	db, err := sql.Open("sqlite", "./db")
	if err != nil {
		log.Fatal(err)
	}
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}
	database = db

	queries := []string{
		"CREATE TABLE IF NOT EXISTS cgo_user (username TEXT, password TEXT)",
		"CREATE TABLE IF NOT EXISTS cgo_product (name TEXT, price INTEGER, description TEXT)",
		"CREATE TABLE IF NOT EXISTS cgo_session (token TEXT, user_id INTEGER)",
		"CREATE TABLE IF NOT EXISTS cgo_cart_item (product_id INTEGER, quantity INTEGER, user_id INTEGER)",
		"CREATE TABLE IF NOT EXISTS cgo_transaction (user_id INTEGER, created_at TEXT)",
		"CREATE TABLE IF NOT EXISTS cgo_line_item (transaction_id INTEGER, product_id INTEGER, quantity INTEGER)",
	}
	for _, q := range queries {
		_, err := db.Exec(q)
		if err != nil {
			log.Fatal(err)
		}
	}

	var q string
	var count int

	q = "SELECT COUNT(*) FROM cgo_user"
	err = db.QueryRow(q).Scan(&count)
	if err != nil {
		log.Fatal(err)
	}
	if count == 0 {
		q = "INSERT INTO cgo_user (username, password) VALUES (?, ?)"
		for _, u := range seedUsers {
			_, err = db.Exec(q, u.Username, u.Password)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	q = "SELECT COUNT(*) FROM cgo_product"
	err = db.QueryRow(q).Scan(&count)
	if err != nil {
		log.Fatal(err)
	}
	if count == 0 {
		q = "INSERT INTO cgo_product (name, price, description) VALUES (?, ?, ?)"
		for _, p := range seedProducts {
			_, err = db.Exec(q, p.Name, p.Price, p.Description)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}

type Product struct {
	Id          int
	Name        string
	Price       int
	Description string
}

func getProducts() []Product {
	var result []Product
	q := "SELECT rowid, name, price, description FROM cgo_product"
	rows, err := database.Query(q)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var product Product
		err = rows.Scan(&product.Id, &product.Name, &product.Price, &product.Description)
		if err != nil {
			log.Fatal(err)
		}
		result = append(result, product)
	}
	return result
}

type User struct {
	Id       int
	Username string
	Password string
}

func getUsers() []User {
	var result []User
	q := "SELECT rowid, username, password FROM cgo_user"
	rows, err := database.Query(q)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var user User
		err = rows.Scan(&user.Id, &user.Username, &user.Password)
		if err != nil {
			log.Fatal(err)
		}
		result = append(result, user)
	}
	return result
}

type Session struct {
	Token  string
	UserId int
}

func setSession(token string, user User) {
	q := "INSERT INTO cgo_session (token, user_id) VALUES (?, ?)"
	_, err := database.Exec(q, token, user.Id)
	if err != nil {
		log.Fatal(err)
	}
}

func getUserFromSessionToken(token string) User {
	q := `
	SELECT
		cgo_user.rowid,
		cgo_user.username,
		cgo_user.password
	FROM cgo_session
	INNER JOIN cgo_user
	ON cgo_session.user_id = cgo_user.rowid
	WHERE cgo_session.token = ?
	LIMIT 1;
	`
	var user User
	err := database.QueryRow(q, token).Scan(&user.Id, &user.Username, &user.Password)
	if err == sql.ErrNoRows {
		return User{}
	} else if err != nil {
		log.Fatal(err)
	}
	return user
}

func createCartItem(userId int, productId int, quantity int) {
	q := "INSERT INTO cgo_cart_item (user_id, product_id, quantity) VALUES (?, ?, ?)"
	_, err := database.Exec(q, userId, productId, quantity)
	if err != nil {
		log.Fatal(err)
	}
}

type CartItem struct {
	Id          int
	UserId      int
	ProductId   int
	Quantity    int
	ProductName string
}

func getCartItemsByUser(user User) []CartItem {
	userId := user.Id
	q := `
	SELECT
		cgo_cart_item.rowid,
		cgo_cart_item.user_id,
		cgo_cart_item.product_id,
		cgo_cart_item.quantity,
		cgo_product.name
	FROM cgo_cart_item
	LEFT JOIN cgo_product ON cgo_cart_item.product_id = cgo_product.rowid
	WHERE cgo_cart_item.user_id = ?
	`
	rows, err := database.Query(q, userId)
	if err == sql.ErrNoRows {
		return []CartItem{}
	} else if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	var result []CartItem
	for rows.Next() {
		var cartItem CartItem
		rows.Scan(&cartItem.Id, &cartItem.UserId, &cartItem.ProductId, &cartItem.Quantity, &cartItem.ProductName)
		result = append(result, cartItem)
	}
	return result
}

func checkoutItemsForUser(user User) {
	cartItems := getCartItemsByUser(user)
	now := time.Now().UTC()
	q := "INSERT INTO cgo_transaction (user_id, created_at) VALUES (?, ?)"
	res, err := database.Exec(q, user.Id, now)
	if err != nil {
		log.Fatal(err)
	}
	lastInsertId, err := res.LastInsertId()
	if err != nil {
		log.Fatal(err)
	}
	for _, ci := range cartItems {
		var q string
		q = "INSERT INTO cgo_line_item (transaction_id, product_id, quantity) VALUES (?, ?, ?)"
		_, err = database.Exec(q, lastInsertId, ci.ProductId, ci.Quantity)
		if err != nil {
			log.Fatal(err)
		}
		q = "DELETE FROM cgo_cart_item WHERE rowid = ?"
		_, err = database.Exec(q, ci.Id)
		if err != nil {
			log.Fatal(err)
		}
	}
}