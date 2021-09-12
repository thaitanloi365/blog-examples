package main

import (
	"database/sql"
	"fmt"
	"golang-gorm-join/db"
	"golang-gorm-join/models"
	"golang-gorm-join/query"
	"golang-gorm-join/query/queryfunc"

	jsoniter "github.com/json-iterator/go"
	"gorm.io/driver/sqlite"
)

func dummyData(db *db.DB) {
	var users []*models.User

	for i := 0; i < 20; i++ {
		users = append(users, &models.User{
			Name: fmt.Sprintf("User %d", i),
			Company: &models.Company{
				Name: fmt.Sprintf("Company %d", i),
			},
			CreditCards: []*models.CreditCard{
				{
					Number: fmt.Sprintf("User %d - Card 1", i),
					CVV:    "111",
				},
				{
					Number: fmt.Sprintf("User %d - Card 2", i),
					CVV:    "2",
				},
			},
		})
	}

	db.Create(&users)
}

func printJSON(v interface{}) {
	data, _ := jsoniter.MarshalIndent(v, "", "    ")
	fmt.Println(string(data))
}

func main() {
	var db = db.New(sqlite.Open("gorm.db"))
	var schemes = []interface{}{
		&models.User{}, models.Company{}, &models.CreditCard{},
	}
	// Cleaning up data
	db.Migrator().DropTable(schemes...)

	db.AutoMigrate(schemes...)
	dummyData(db)

	// Test pagination
	var paginationResult = query.NewQueryBuilder(db, queryfunc.NewUserBuilder()).Limit(20).PagingFunc()
	printJSON(paginationResult)

	// Test find func
	var users []*models.User
	var err = query.NewQueryBuilder(db, queryfunc.NewUserBuilder()).
		WhereFunc(func(builder *query.Builder) {

			builder.Where("u.name = ?", "User 1")

			builder.Where("u.name = @name", sql.Named("name", "User 1"))

		}).
		FindFunc(&users)
	if err != nil {
		panic(err)
	}
	printJSON(users)

}
