package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

var db *sqlx.DB

type Person struct {
	ID          int    `json:"id" db:"id"`
	Name        string `json:"name" db:"name"`
	Age         int    `json:"age" db:"age"`
	PhoneNumber string `json:"phone_number" db:"phone_number"`
	City        string `json:"city" db:"city"`
	State       string `json:"state" db:"state"`
	Street1     string `json:"street1" db:"street1"`
	Street2     string `json:"street2" db:"street2"`
	ZipCode     string `json:"zip_code" db:"zip_code"`
}

func initDB() {
	var err error
	dsn := "root:dhiraj@1999@tcp(127.0.0.1:3306)/cetec?charset=utf8&parseTime=True&loc=Local"
	db, err = sqlx.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatalf("Error pinging database: %v", err)
	}
}

func getPersonInfo(c *gin.Context) {
	personID := c.Param("person_id")
	var person Person

	query := `
SELECT p.id, p.name, p.age, ph.number AS phone_number, a.city, a.state, a.street1, a.street2, a.zip_code
FROM person p
JOIN phone ph ON ph.person_id = p.id
JOIN address_join aj ON aj.person_id = p.id
JOIN address a ON a.id = aj.address_id
WHERE p.id = ?
`

	err := db.Get(&person, query, personID)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no record found"})
		return
	}

	c.JSON(http.StatusOK, person)
}

func createPerson(c *gin.Context) {
	var newPerson Person
	if err := c.ShouldBindJSON(&newPerson); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data"})
		return
	}

	tx, err := db.Beginx()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error starting transaction"})
		return
	}

	// Insert new person
	insertPersonQuery := `INSERT INTO person(name, age) VALUES (?, ?)`
	result, err := tx.Exec(insertPersonQuery, newPerson.Name, newPerson.Age)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error inserting person"})
		return
	}

	// Get the person ID
	personID, err := result.LastInsertId()
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting person ID"})
		return
	}

	// Insert phone number
	insertPhoneQuery := `INSERT INTO phone(person_id, number) VALUES (?, ?)`
	_, err = tx.Exec(insertPhoneQuery, personID, newPerson.PhoneNumber)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error inserting phone number"})
		return
	}

	// Insert address
	insertAddressQuery := `INSERT INTO address(city, state, street1, street2, zip_code) VALUES (?, ?, ?, ?, ?)`
	addressResult, err := tx.Exec(insertAddressQuery, newPerson.City, newPerson.State, newPerson.Street1, newPerson.Street2, newPerson.ZipCode)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error inserting address"})
		return
	}

	// Get the address ID
	addressID, err := addressResult.LastInsertId()
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting address ID"})
		return
	}

	// Insert into address_join table to link person and address
	insertAddressJoinQuery := `INSERT INTO address_join(person_id, address_id) VALUES (?, ?)`
	_, err = tx.Exec(insertAddressJoinQuery, personID, addressID)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error linking person to address"})
		return
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error committing transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Person created successfully"})
}

func main() {
	initDB()

	r := gin.Default()

	r.GET("/person/:person_id/info", getPersonInfo)
	r.POST("/person/create", createPerson)

	r.Run(":8080")
}
