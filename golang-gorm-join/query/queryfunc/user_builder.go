package queryfunc

import (
	"golang-gorm-join/db"
	"golang-gorm-join/models"
)

type userCopy struct {
	*models.User

	Company     *models.Company      `gorm:"embedded;embeddedPrefix:c__"`
	CreditCards []*models.CreditCard `gorm:"-" json:"credit_cards"`
}

func NewUserBuilder() *Builder {

	const (
		rawSQL = `
		SELECT u.*,
		c.id AS c__id,
		c.updated_at AS c__updated_at,
		c.deleted_at AS c__deleted_at,
		c.name AS c__name

		FROM users u
		LEFT JOIN companies c ON c.id = u.company_id
		`
		countSQL = `
		SELECT 1 FROM users u
		LEFT JOIN companies c ON c.id = u.company_id
		`
	)

	return NewBuilder(rawSQL, countSQL).
		WithPaginationFunc(func(db, rawSQL *db.DB) (interface{}, error) {
			var records = make([]*models.User, rawSQL.RowsAffected)
			rows, err := rawSQL.Rows()
			if err != nil {
				return nil, err
			}
			defer rows.Close()

			var userIDs []uint
			for rows.Next() {
				var copy userCopy
				err = db.ScanRows(rows, &copy)
				if err != nil {
					continue
				}

				// Copy to original model
				copy.User.Company = copy.Company

				// Copy pk key for other join
				userIDs = append(userIDs, copy.User.ID)

				records = append(records, copy.User)
			}

			var creditCards []*models.CreditCard
			if err := db.Find(&creditCards, "user_id IN ?", userIDs).Error; err == nil {
				for _, user := range records {
					for _, cc := range creditCards {
						if cc.UserID == user.ID {
							user.CreditCards = append(user.CreditCards, cc)
						}
					}
				}
			}

			return &records, err
		})
}
