package hdv

import (
	"dna"
	"dna/sqlpg"
	"dna/terminal"
	"errors"
	"fmt"
	"reflect"
	"time"
)

func SaveLastestMovieCurrentEps(db *sqlpg.DB, tblName dna.String, logger *terminal.Logger) {
	for mvid, currentEp := range LastestMovieCurrentEps {
		query := "UPDATE " + tblName + " SET current_eps=" + currentEp.ToString()
		query += " WHERE id=" + mvid.ToString() + ";"
		_, err := db.Exec(query.String())
		if err != nil {
			logger.Println("$$$error$$$" + query + "$$$error$$$")
		}
	}
}

// GetMoviesCurrentEps returns a map of MovideId and CurrentEps
// if CurrentEps is less than MaxEp.
// it returns an error if available.
//
// This function is used when we need to find all possible movie ids
// to update.
func GetMoviesCurrentEps(db *sqlpg.DB, tblName dna.String) (map[dna.Int]dna.Int, error) {
	var movieCurrentEps = make(map[dna.Int]dna.Int)
	ids := &[]dna.Int{}
	currentEps := &[]dna.Int{}
	err := db.Select(ids, dna.Sprintf(`SELECT id from %v where current_eps < max_ep order by id DESC`, tblName))
	if err != nil {
		return nil, err
	}
	err = db.Select(currentEps, dna.Sprintf(`SELECT current_eps from %v where current_eps < max_ep order by id DESC`, tblName))
	if err != nil {
		return nil, err
	}
	if len(*currentEps) != len(*ids) {
		return nil, errors.New("Length of IDs and CurrentEps is not correspondent")
	}
	for idx, movieid := range *ids {
		movieCurrentEps[movieid] = (*currentEps)[idx]
	}
	return movieCurrentEps, nil
}

// splitAndTruncateArtists splits stringarray by the key "feat:"
// and filter only string elements not equal to empty string.
func splitAndTruncateArtists(artists dna.StringArray) dna.StringArray {
	return dna.StringArray(artists.SplitWithRegexp("feat:").Map(func(val dna.String, idx dna.Int) dna.String {
		return val.Trim()
	}).([]dna.String)).Filter(func(val dna.String, idx dna.Int) dna.Bool {
		if val != "" {
			return true
		} else {
			return false
		}
	})
}

func getColumn(f reflect.StructField, structValue interface{}) (dna.String, dna.String) {
	var columnName, columnValue dna.String
	switch f.Type.String() {
	case "dna.Int":
		columnName = dna.String(f.Name).ToSnakeCase()
		columnValue = dna.String(fmt.Sprintf("%v", structValue))

	case "dna.Float":
		columnName = dna.String(f.Name).ToSnakeCase()
		columnValue = dna.String(fmt.Sprintf("%v", structValue))

	case "dna.Bool":
		columnName = dna.String(f.Name).ToSnakeCase()
		columnValue = dna.String(fmt.Sprintf("%v", structValue))

	case "dna.String":
		columnName = dna.String(f.Name).ToSnakeCase()
		columnValue = dna.String(fmt.Sprintf("$binhdna$%v$binhdna$", structValue))

	case "dna.StringArray":
		var tempStr dna.String = dna.String(fmt.Sprintf("%#v", structValue)).Replace("dna.StringArray", "")
		columnName = dna.String(f.Name).ToSnakeCase()
		columnValue = dna.String(fmt.Sprintf("$binhdna$%v$binhdna$", tempStr))

	case "dna.IntArray":
		var tempStr dna.String = dna.String(fmt.Sprintf("%#v", structValue)).Replace("dna.IntArray", "")
		columnName = dna.String(f.Name).ToSnakeCase()
		columnValue = dna.String(fmt.Sprintf("$binhdna$%v$binhdna$", tempStr))
	case "time.Time":
		columnName = dna.String(f.Name).ToSnakeCase()
		datetime := structValue.(time.Time)
		if !datetime.IsZero() {
			columnValue = dna.String(fmt.Sprintf("$binhdna$%v$binhdna$", dna.String(datetime.String()).ReplaceWithRegexp(`\+.+$`, ``).Trim()))
		} else {
			columnValue = dna.String(fmt.Sprintf("%v", "NULL"))
		}

	default:
		// panic("A Field of struct is not dna basic type")
	}
	return columnName, columnValue
}

func getInsertStatement(tbName dna.String, structValue interface{}, condStr dna.String, isPrintable dna.Bool) dna.String {
	var realKind string
	var columnNames, columnValues dna.StringArray
	tempintslice := []int{0}
	var ielements int
	var kind string = reflect.TypeOf(structValue).Kind().String()
	if kind == "ptr" {
		realKind = reflect.TypeOf(structValue).Elem().Kind().String()

	} else {
		realKind = reflect.TypeOf(structValue).Kind().String()

	}

	if realKind != "struct" {
		panic("Param has to be struct")
	}

	if kind == "ptr" {
		ielements = reflect.TypeOf(structValue).Elem().NumField()
	} else {
		ielements = reflect.TypeOf(structValue).NumField()
	}

	for i := 0; i < ielements; i++ {
		tempintslice[0] = i
		if kind == "ptr" {
			f := reflect.TypeOf(structValue).Elem().FieldByIndex(tempintslice)
			v := reflect.ValueOf(structValue).Elem().FieldByIndex(tempintslice)
			clName, clValue := getColumn(f, v.Interface())
			columnNames.Push(clName)
			columnValues.Push(clValue)
		} else {
			f := reflect.TypeOf(structValue).FieldByIndex(tempintslice)
			v := reflect.ValueOf(structValue).FieldByIndex(tempintslice)
			clName, clValue := getColumn(f, v.Interface())
			columnNames.Push(clName)
			columnValues.Push(clValue)
		}

	}
	if isPrintable == true {
		return "INSERT INTO " + tbName + "\n(" + columnNames.Join(",") + ")\n" + " SELECT " + columnValues.Join(",\n") + " \n" + condStr
	} else {
		return "INSERT INTO " + tbName + "(" + columnNames.Join(",") + ")" + " SELECT " + columnValues.Join(",") + " " + condStr
	}
}

// GetTableName returns table name from a struct.
// Ex: An instance of ns.Song will return nssongs
// An instance of ns.Album will return nsalbums
func getTableName(structValue interface{}) dna.String {
	val := reflect.TypeOf(structValue)
	if val.Kind() != reflect.Ptr {
		panic("StructValue has to be pointer")
		if val.Elem().Kind() != reflect.Struct {
			panic("StructValue has to be struct type")
		}
	}
	return dna.String(val.Elem().String()).Replace(".", "").ToLowerCase() + "s"
}

func getInsertStmt(structValue interface{}, condStr dna.String) dna.String {
	return getInsertStatement(getTableName(structValue), structValue, condStr, false) + ";"
}