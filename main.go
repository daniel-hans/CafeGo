package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type IndexPageData struct {
	Username string
	Products []Product
}

type CartPageData struct {
	CartItems []CartItem
	User      User
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("./templates/index.html")
	if err != nil {
		log.Fatal(err)
	}

	cookies := r.Cookies()
	var sessionToken string
	for _, cookie := range cookies {
		if cookie.Name == "cafego_session" {
			sessionToken = cookie.Value
			break
		}
	}
	user := getUserFromSessionToken(sessionToken)

	sampleProducts := getProducts()

	samplePageData := IndexPageData{
		Username: user.Username,
		Products: sampleProducts,
	}

	err = tmpl.Execute(w, samplePageData)
	if err != nil {
		log.Fatal(err)
	}
}

func productHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		reqPath := r.URL.Path
		splitPath := strings.Split(reqPath, "/")
		elemCount := len(splitPath)
		productId := splitPath[elemCount-1]

		intId, err := strconv.Atoi(productId)
		if err != nil {
			log.Fatal(err)
		}

		var product Product
		for _, p := range getProducts() {
			if p.Id == intId {
				product = p
				break
			}
		}

		if product == (Product{}) {
			log.Fatal("Can't find product with that ID")
		}

		tmpl, err := template.ParseFiles("./templates/product.html")
		if err != nil {
			log.Fatal(err)
		}
		err = tmpl.Execute(w, product)
		if err != nil {
			log.Fatal(err)
		}
	} else if r.Method == "POST" {
		cookies := r.Cookies()
		var sessionToken string
		for _, cookie := range cookies {
			if cookie.Name == "cafego_session" {
				sessionToken = cookie.Value
				break
			}
		}
		user := getUserFromSessionToken(sessionToken)
		userId := user.Id

		sProductId := r.FormValue("product_id")
		productId, err := strconv.Atoi(sProductId)
		if err != nil {
			log.Fatal(err)
		}

		sQuantity := r.FormValue("quantity")
		quantity, err := strconv.Atoi(sQuantity)
		if err != nil {
			log.Fatal(err)
		}

		createCartItem(userId, productId, quantity)
		http.Redirect(w, r, "/", http.StatusFound)
	}
}

func generateSessionToken() string {
	rawBytes := make([]byte, 16)
	_, err := rand.Read(rawBytes)
	if err != nil {
		log.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(rawBytes)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		tmpl, err := template.ParseFiles("./templates/login.html")
		if err != nil {
			log.Fatal(err)
		}
		err = tmpl.Execute(w, nil)
		if err != nil {
			log.Fatal(err)
		}
	} else if r.Method == "POST" {
		rUsername := r.FormValue("username")
		rPassword := r.FormValue("password")

		var user User
		for _, u := range getUsers() {
			if (rUsername == u.Username) && (rPassword == u.Password) {
				user = u
			}
		}

		if user == (User{}) {
			fmt.Fprint(w, "Invalid login. Please go back and try again.")
			return
		}

		token := generateSessionToken()
		setSession(token, user)

		cookie := http.Cookie{Name: "cafego_session", Value: token, Path: "/"}
		http.SetCookie(w, &cookie)

		http.Redirect(w, r, "/", http.StatusFound)
	}
}

func cartHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		tmpl, err := template.ParseFiles("./templates/cart.html")
		if err != nil {
			log.Fatal(err)
		}

		cookies := r.Cookies()
		var sessionToken string
		for _, cookie := range cookies {
			if cookie.Name == "cafego_session" {
				sessionToken = cookie.Value
				break
			}
		}
		user := getUserFromSessionToken(sessionToken)

		cartItems := getCartItemsByUser(user)

		pageData := CartPageData{
			CartItems: cartItems,
			User:      user,
		}
		tmpl.Execute(w, pageData)
	} else if r.Method == "POST" {
		cookies := r.Cookies()
		var sessionToken string
		for _, cookie := range cookies {
			if cookie.Name == "cafego_session" {
				sessionToken = cookie.Value
				break
			}
		}
		user := getUserFromSessionToken(sessionToken)

		checkoutItemsForUser(user)

		http.Redirect(w, r, "/", http.StatusFound)
	}
}

func main() {
	initDB()
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/product/", productHandler)
	http.HandleFunc("/login/", loginHandler)
	http.HandleFunc("/cart/", cartHandler)
	http.ListenAndServe(":5000", nil)
}

//.\sqlite3 db
//SELECT * FROM cgo_line_item;