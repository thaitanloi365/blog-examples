package query

import (
	"database/sql"
	"fmt"
	"golang-gorm-join/db"
	"log"
	"math"
	"reflect"
	"strings"

	"gorm.io/gorm"
)

type QueryBuilder interface {
	GetRawSQL() string
	GetCountRawSQL() string
	GetGroupByRawSQL() string
	GetOrderByRawSQL() string
	GetPaginationFunc() ExecFunc
	IsWrapJSON() bool
}

// ExecFunc exec func
type ExecFunc = func(db *db.DB, rawSQL *db.DB) (interface{}, error)

// WhereFunc where func
type WhereFunc = func(builder *Builder)

// Pagination ...
type Pagination struct {
	HasNext     bool        `json:"has_next"`
	HasPrev     bool        `json:"has_prev"`
	PerPage     int         `json:"per_page"`
	NextPage    int         `json:"next_page"`
	Page        int         `json:"current_page"`
	PrevPage    int         `json:"prev_page"`
	Offset      int         `json:"offset"`
	Records     interface{} `json:"records"`
	TotalRecord int         `json:"total_record"`
	TotalPage   int         `json:"total_page"`
	Metadata    interface{} `json:"metadata"`
}

// Builder query config
type Builder struct {
	db               *db.DB
	RawSQLString     string
	countRawSQL      string
	limit            int
	page             int
	hasWhere         bool
	whereValues      []interface{}
	namedWhereValues map[string]interface{}
	orderBy          string
	groupBy          string
	wrapJSON         bool
	qf               QueryBuilder
}

// New init
func New(db *db.DB, rawSQL string, countSQL ...string) *Builder {
	var builder = &Builder{
		db:               db,
		RawSQLString:     rawSQL,
		whereValues:      []interface{}{},
		namedWhereValues: map[string]interface{}{},
		hasWhere:         false,
		orderBy:          "",
		groupBy:          "",
		wrapJSON:         false,
	}
	if len(countSQL) > 0 {
		builder.countRawSQL = countSQL[0]
	}

	return builder
}

func NewQueryBuilder(db *db.DB, qf QueryBuilder) *Builder {
	var builder = &Builder{
		db:               db,
		RawSQLString:     qf.GetRawSQL(),
		whereValues:      []interface{}{},
		namedWhereValues: map[string]interface{}{},
		hasWhere:         false,
		orderBy:          qf.GetOrderByRawSQL(),
		groupBy:          qf.GetGroupByRawSQL(),
		wrapJSON:         qf.IsWrapJSON(),
		countRawSQL:      qf.GetCountRawSQL(),
		qf:               qf,
	}

	return builder
}

func (b *Builder) WithWrapJSON(isWrapJSON bool) *Builder {
	b.wrapJSON = isWrapJSON
	return b
}

func (b *Builder) Where(query interface{}, args ...interface{}) *Builder {
	switch value := query.(type) {
	case map[string]interface{}:
		for key, v := range value {
			b.namedWhereValues[key] = v
		}
	case map[string]string:
		for key, v := range value {
			b.namedWhereValues[key] = v
		}
	case sql.NamedArg:
		b.namedWhereValues[value.Name] = value.Value
	default:
		if len(args) > 0 {
			b.whereValues = append(b.whereValues, args...)
		}

		if b.hasWhere {
			b.RawSQLString = fmt.Sprintf("%s AND %v", b.RawSQLString, query)
			if b.countRawSQL != "" {
				b.countRawSQL = fmt.Sprintf("%s AND %v", b.countRawSQL, query)
			}
		} else {
			b.RawSQLString = fmt.Sprintf("%s WHERE %v", b.RawSQLString, query)
			if b.countRawSQL != "" {
				b.countRawSQL = fmt.Sprintf("%s WHERE %v", b.countRawSQL, query)
			}
			b.hasWhere = true

		}

	}

	return b
}

// OrderBy specify order when retrieve records from database
func (b *Builder) OrderBy(orderBy ...string) *Builder {
	if len(orderBy) > 0 {
		b.orderBy = strings.Join(orderBy, ",")
	}
	return b
}

// GroupBy specify the group method on the find
func (b *Builder) GroupBy(groupBy string) *Builder {
	b.groupBy = groupBy
	return b
}

// WhereFunc using where func
func (b *Builder) WhereFunc(f WhereFunc) *Builder {
	f(b)
	return b
}

// Limit limit
func (b *Builder) Limit(limit int) *Builder {
	b.limit = limit
	return b
}

// Page offset
func (b *Builder) Page(page int) *Builder {
	b.page = page
	return b
}

// Build build
func (b *Builder) build() (queryString string, countQuery string) {
	var rawSQLString = b.RawSQLString
	queryString = rawSQLString
	countQuery = b.countRawSQL

	if countQuery == "" {
		countQuery = rawSQLString
	}

	if b.groupBy != "" {
		queryString = fmt.Sprintf("%s GROUP BY %s", queryString, b.groupBy)
		countQuery = fmt.Sprintf("%s GROUP BY %s", countQuery, b.groupBy)
	}

	if b.orderBy != "" {
		queryString = fmt.Sprintf("%s ORDER BY %s", queryString, b.orderBy)
	}

	if b.limit > 0 {
		queryString = fmt.Sprintf("%s LIMIT %d", queryString, b.limit)
	}

	if b.page > 0 {
		var offset = 0
		if b.page > 1 {
			offset = (b.page - 1) * b.limit
		}

		queryString = fmt.Sprintf("%s OFFSET %d", queryString, offset)
	}

	if b.wrapJSON {
		queryString = fmt.Sprintf(`
WITH alias AS (
%s
)
SELECT to_jsonb(row_to_json(alias)) AS alias
FROM alias
		`, queryString)
	}

	return
}

func (b *Builder) GetPagingFunc(f ...ExecFunc) ExecFunc {
	if b.qf != nil {
		return b.qf.GetPaginationFunc()
	}

	if len(f) > 0 {
		return f[0]
	}

	return nil
}

// PagingFunc paging
func (b *Builder) PagingFunc(f ...ExecFunc) *Pagination {
	if b.page < 1 {
		b.page = 1
	}
	var fn = b.GetPagingFunc(f...)
	if fn == nil {
		panic(fmt.Errorf("fn is not implement"))
	}

	var offset = (b.page - 1) * b.limit
	var done = make(chan bool, 1)
	var pagination Pagination
	var count int

	sqlString, countSQLString := b.build()

	var values = b.mergeValues()
	countSQLString = fmt.Sprintf(`
SELECT COUNT(1) 
FROM (
%s
) t
	`, countSQLString)
	var countSQL = b.db.WithGorm(b.db.Raw(countSQLString, values...))
	go b.count(countSQL, done, &count)

	result, err := fn(b.db, b.db.WithGorm(b.db.Raw(sqlString, values...)))
	if err != nil {
		log.Fatalln(err)
	}
	<-done
	close(done)

	pagination.TotalRecord = count
	pagination.Records = result
	pagination.Page = b.page
	pagination.Offset = offset

	if b.limit > 0 {
		pagination.PerPage = b.limit
		pagination.TotalPage = int(math.Ceil(float64(count) / float64(b.limit)))
	} else {
		pagination.TotalPage = 1
		pagination.PerPage = count
	}

	if b.page > 1 {
		pagination.PrevPage = b.page - 1
	} else {
		pagination.PrevPage = b.page
	}

	if b.page == pagination.TotalPage {
		pagination.NextPage = b.page
	} else {
		pagination.NextPage = b.page + 1
	}

	pagination.HasNext = pagination.TotalPage > pagination.Page
	pagination.HasPrev = pagination.Page > 1

	if !pagination.HasNext {
		pagination.NextPage = pagination.Page
	}

	return &pagination
}

func (b *Builder) FindFunc(dest interface{}, f ...ExecFunc) error {
	sqlString, _ := b.build()

	var rOut = reflect.ValueOf(dest)
	if rOut.Kind() != reflect.Ptr {
		return fmt.Errorf("must be a pointer of %T", dest)
	}

	var fn = b.GetPagingFunc(f...)
	if fn == nil {
		panic(fmt.Errorf("fn is not implement"))
	}

	var values = b.mergeValues()
	result, err := fn(b.db, b.db.WithGorm(b.db.Raw(sqlString, values...)))
	if err != nil {
		return err
	}

	return b.copyResult(rOut, result)
}

// Scan scan
func (b *Builder) FirstFunc(dest interface{}, f ...ExecFunc) error {
	b.limit = 1
	sqlString, _ := b.build()

	var rOut = reflect.ValueOf(dest)
	if rOut.Kind() != reflect.Ptr {
		return fmt.Errorf("must be a pointer of %T", dest)
	}

	var fn = b.GetPagingFunc(f...)
	if fn == nil {
		panic(fmt.Errorf("fn is not implement"))
	}

	var values = b.mergeValues()
	result, err := fn(b.db, b.db.WithGorm(b.db.Raw(sqlString, values...)))
	if err != nil {
		return err
	}
	return b.copyResult(rOut, result)
}

// Scan scan
func (b *Builder) Scan(dest interface{}) error {
	sqlString, _ := b.build()

	var values = b.mergeValues()
	var result = b.db.Raw(sqlString, values...).Scan(dest)
	if result.Error != nil {
		if result.RowsAffected == 0 {
			return sql.ErrNoRows
		}
	}

	return result.Error
}

func (b *Builder) Find(dest interface{}) error {
	sqlString, _ := b.build()

	var values = b.mergeValues()
	var result = b.db.Raw(sqlString, values...).Find(dest)
	if result.Error != nil {
		if result.RowsAffected == 0 {
			return sql.ErrNoRows
		}
	}

	return result.Error
}

func (b *Builder) ExplainSQL() string {
	sqlString, _ := b.build()

	var values = b.mergeValues()
	var stmt = b.db.Session(&gorm.Session{DryRun: true}).Raw(sqlString, values...).Statement
	return stmt.Explain(stmt.SQL.String(), stmt.Vars...)

}

// ScanRow scan
func (b *Builder) ScanRow(dest interface{}) error {
	sqlString, _ := b.build()

	var values = b.mergeValues()
	var err = b.db.Raw(sqlString, values).Row().Scan(dest)
	if err != nil {
		log.Fatalln(err)
		return err
	}

	return nil
}

// PrepareCountSQL prepare statement
func (b *Builder) count(countSQL *db.DB, done chan bool, count *int) {
	if countSQL != nil {
		countSQL.Row().Scan(count)
	}
	done <- true
}

// PrepareCountSQL prepare statement
func (b *Builder) mergeValues() []interface{} {
	var values = []interface{}{}
	values = append(values, b.whereValues...)
	values = append(values, b.namedWhereValues)
	return values
}

func (b *Builder) copyResult(rOut reflect.Value, result interface{}) error {
	var rResult = reflect.ValueOf(result)

	if rResult.Kind() != reflect.Ptr {
		rResult = toPtr(rResult)

	}

	if rResult.Type() != rOut.Type() {
		switch rResult.Kind() {
		case reflect.Array, reflect.Slice:
			if rResult.Len() > 0 {
				var elem = rResult.Index(0).Elem()
				rOut.Elem().Set(elem)
				return nil
			} else {
				return sql.ErrNoRows
			}
		case reflect.Ptr:
			switch rResult.Elem().Kind() {
			case reflect.Array, reflect.Slice:
				if rResult.Elem().Len() > 0 {
					var elem = rResult.Elem().Index(0).Elem()
					rOut.Elem().Set(elem)
					return nil
				} else {
					return sql.ErrNoRows
				}
			}
		}

		return fmt.Errorf("%v is not %v", rResult.Type(), rOut.Type())
	}

	rOut.Elem().Set(rResult.Elem())

	return nil
}

// ToPtr wraps the given value with pointer: V => *V, *V => **V, etc.
func toPtr(v reflect.Value) reflect.Value {
	pt := reflect.PtrTo(v.Type()) // create a *T type.
	pv := reflect.New(pt.Elem())  // create a reflect.Value of type *T.
	pv.Elem().Set(v)              // sets pv to point to underlying value of v.
	return pv
}
