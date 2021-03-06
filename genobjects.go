package main

import (
    "fmt"
    "github.com/network4all/mydb"
    "github.com/network4all/configuration"
    "github.com/network4all/logerror"
    "time"
    "unicode"
    "strings"
    "strconv"
    "os"
    "flag"
)

const (
   ws = "%s"
)

var conf configuration.Settings
var reservedWords map[string]string


func usage() {
    fmt.Fprintf(os.Stderr, "usage: myprog [json.conf]\n")
    flag.PrintDefaults()
    os.Exit(2)
}

func main() {

/*
   flag.Usage = usage
   flag.Parse()

   args := flag.Args()

   if len(args) > 0 {
      dbconfigpath := args[0] 
      //reinit db
      configuration.LoadsettingsPfn(&conf, dbconfigpath)
      mydb.InitConfigSettings(conf)
   } 
*/

   currenttime := time.Now().Local()

   reservedWords = map[string]string {
      "id": "1",
      "enabled": "1",
      "lastupdate": "1",
   }

   fmt.Printf("package %sobjects\n\n", conf.DBname)
   fmt.Printf("// DOC: generated by db2object generator on %s\n\n", currenttime.Format("2006-01-02 15:04:05 +0800"))
   // import
   fmt.Printf("// import\n")
   fmt.Printf("import (\n")
   fmt.Printf("   \"fmt\"\n")
   fmt.Printf("   \"github.com/network4all/mydb\"\n")
   fmt.Printf("   \"github.com/network4all/logerror\"\n")
   fmt.Printf("   _ \"strconv\"\n")
   fmt.Printf("   \"time\"\n")
   fmt.Printf(")\n\n")

   // structs
   genStruct()
   genCrud()

   close()
}

func genCrud() {
   queryTables := fmt.Sprintf("SHOW TABLES")
   rowTables, err := mydb.DB.Query(queryTables)
   logerror.CheckErr(err)

   // for each table
   for rowTables.Next() {
      var tablename string
      err = rowTables.Scan(&tablename)
      logerror.CheckErr(err)
      genLoadObjects(tablename)
      genParentChild(tablename)
      genChildParent(tablename)
      genChildCount(tablename)
      genLoadById(tablename)
      genFilterCount(tablename)
      genSave(tablename)
      genUpdate(tablename)
      genDisable(tablename)
      genEnable(tablename)
      genTruncate(tablename)
      genGetID(tablename)
      genCollisionDetect(tablename)
   }
}

func genLoadById(tablename string ) {

   var fieldsComma = ""    // id, name, ipadres, description"
   var fieldsAnd = ""      // &id, &name, &ipadres, &description"
   var fieldsAssign = ""   // Id:id, Name:name, Ipadres:ipadres, Description:description"
   var fieldsVarList = ""  // var id int\n      var ipadres string\n      var name string\n      var description string\n"

   // for each field
   queryTables := fmt.Sprintf("SELECT COLUMN_NAME, DATA_TYPE FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = '%s' AND TABLE_NAME = '%s';", conf.DBname, tablename)
   rowFields, err := mydb.DB.Query(queryTables)
   for rowFields.Next() {
      var FieldName string
      var DATA_TYPE string
      err = rowFields.Scan(&FieldName, &DATA_TYPE)
      logerror.CheckErr(err)
      fieldsComma = fmt.Sprintf("%s%s, ", fieldsComma, sqlfieldname(FieldName))
      fieldsAnd = fmt.Sprintf("%s&%s, ", fieldsAnd, dbfieldname(FieldName))
      fieldsAssign = fmt.Sprintf("%s%s:%s, ", fieldsAssign, dbfieldname(FieldName), dbfieldname(FieldName))
      fieldsVarList = fmt.Sprintf("%s      var %s %s\n", fieldsVarList, dbfieldname(FieldName), DataTypeConversion(DATA_TYPE))
   }

   // trim comma's
   fieldsComma = fmt.Sprintf("%s", TrimSuffix(fieldsComma,", "))
   fieldsAnd = fmt.Sprintf("%s", TrimSuffix(fieldsAnd,", "))
   fieldsAssign = fmt.Sprintf("%s", TrimSuffix(fieldsAssign,", "))

   // object naming
   var object = createSingleObjectName(tablename)

   fmt.Printf("func Load%sbyId(id int) %s {\n", object, object)
   fmt.Printf("   query := fmt.Sprintf(\"SELECT %s FROM %s WHERE id=%s\", id)\n", fieldsComma, tablename, "%d")
   fmt.Printf("   rows, err := mydb.DB.Query(query)\n")
   fmt.Printf("   logerror.CheckErr(err)\n")
   fmt.Printf("   defer rows.Close()\n")
   fmt.Printf("   var myobj %s\n", object)
   fmt.Printf("   if rows.Next() {\n")
   // var list
   fmt.Printf("%s", fieldsVarList)
   fmt.Printf("      err = rows.Scan(%s)\n", fieldsAnd)
   fmt.Printf("      logerror.CheckErr(err)\n")
   fmt.Printf("      myobj =  %s{%s}\n", object, fieldsAssign)
   fmt.Printf("   }\n")
   fmt.Printf("   return myobj\n")
   fmt.Printf("}\n\n")

}

func genParentChild(tablename string) {
   queryTables := fmt.Sprintf("SELECT COLUMN_NAME FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = '%s' AND TABLE_NAME = '%s';", conf.DBname, tablename)
   rowFields, err := mydb.DB.Query(queryTables)
   for rowFields.Next() {
      var FieldName string
      err = rowFields.Scan(&FieldName)
      logerror.CheckErr(err)
      // AKU: todo: nice solution while reading fk_ relations table.
      if strings.HasPrefix(FieldName, "fk_") {
         parentName := FirstLetterUpcase(strings.Replace(FieldName,"fk_","", -1))
         childName := FirstLetterUpcase(tablename)
         fmt.Printf("func (ParentObject *%s) Get%s(ChildCollection *[]%s) {\n", TrimSuffix(parentName,"s"), childName, TrimSuffix(childName,"s"))
         fmt.Printf("   Load%s(ChildCollection, fmt.Sprintf(\"WHERE %s=%s AND enabled=1\", ParentObject.Id))\n", childName, FieldName, "%d")
         fmt.Printf("}\n\n")
      }
   }
}

func genChildParent(tablename string) {
   queryTables := fmt.Sprintf("SELECT COLUMN_NAME FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = '%s' AND TABLE_NAME = '%s';", conf.DBname, tablename)
   rowFields, err := mydb.DB.Query(queryTables)
   for rowFields.Next() {
      var FieldName string
      err = rowFields.Scan(&FieldName)
      logerror.CheckErr(err)
      // AKU: todo: nice solution while reading fk_ relations table.
      if strings.HasPrefix(FieldName, "fk_") {
         parentName := FirstLetterUpcase(strings.Replace(FieldName,"fk_","", -1))
         var parentObject = createSingleObjectName(parentName)
         childObject := FirstLetterUpcase(tablename)

         fmt.Printf("func (childObject *%s) GetParent%s() %s {\n", TrimSuffix(childObject,"s"), TrimSuffix(parentName,"s"), parentObject)
         fmt.Printf("   return Load%sbyId(%s)\n", parentObject, "childObject.Fk_" + strings.ToLower(parentName))
         fmt.Printf("}\n\n")
      }
   }
}

func genChildCount(tablename string) {
   queryTables := fmt.Sprintf("SELECT COLUMN_NAME FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = '%s' AND TABLE_NAME = '%s';", conf.DBname, tablename)
   rowFields, err := mydb.DB.Query(queryTables)
   for rowFields.Next() {
      var FieldName string
      err = rowFields.Scan(&FieldName)
      logerror.CheckErr(err)

      if strings.HasPrefix(FieldName, "fk_") {
         parentName := FirstLetterUpcase(strings.Replace(FieldName,"fk_","", -1))
         parentObjName := createSingleObjectName(parentName)
         childName := createSingleObjectName(tablename)

         fmt.Printf("func (parentObject *%s) %sCount() int {\n", parentObjName, childName)
         fmt.Printf("    query := fmt.Sprintf(\"SELECT count(*) FROM %s WHERE Fk_%s=%s AND enabled=1 ORDER BY name;\", parentObject.Id)\n", tablename, parentName, "%d")
         fmt.Printf("    var count int\n")
         fmt.Printf("    err := mydb.DB.QueryRow(query).Scan(&count)\n")
         fmt.Printf("    logerror.CheckErr(err)\n")
         fmt.Printf("    return count\n")
         fmt.Printf("}\n\n")
      }
   }
}

func genFilterCount(tablename string) {
         childName := createSingleObjectName(tablename)
         fmt.Printf("func %sCount(filter string) int {\n", childName)
         fmt.Printf("    query := fmt.Sprintf(\"SELECT count(*) FROM %s %s;\", filter)\n", tablename, "%s")
         fmt.Printf("    var count int\n")
         fmt.Printf("    err := mydb.DB.QueryRow(query).Scan(&count)\n")
         fmt.Printf("    logerror.CheckErr(err)\n")
         fmt.Printf("    return count\n")
         fmt.Printf("}\n\n")
}

func genTruncate(tablename string) {
         childName := createSingleObjectName(tablename)
         fmt.Printf("func Truncate%ssTable() {\n", childName)
         fmt.Printf("   query := \"TRUNCATE TABLE %s;\"\n", tablename)
         fmt.Printf("   instcmd, err := mydb.DB.Prepare (query)\n")
         fmt.Printf("   logerror.CheckErr(err)\n")
         fmt.Printf("   defer instcmd.Close()\n")
         fmt.Printf("   instcmd.Exec()\n")
         fmt.Printf("}\n\n")
}

func genGetID(tablename string) {
         childName := createSingleObjectName(tablename)
         fmt.Printf("func Get%sId(filter string) int {\n", childName)
         fmt.Printf("    query := fmt.Sprintf(\"SELECT id FROM %s %s;\", filter)\n", tablename, "%s")
         fmt.Printf("    var count int\n")
         fmt.Printf("    err := mydb.DB.QueryRow(query).Scan(&count)\n")
         fmt.Printf("    logerror.CheckErr(err)\n")
         fmt.Printf("    return count\n")
         fmt.Printf("}\n\n")
}

func genCollisionDetect(tablename string) {
   childName := createSingleObjectName(tablename)

   fmt.Printf("func (childObject *%s) Checkcollision() bool {\n", childName)
   fmt.Printf("   dummy := Load%sbyId(childObject.Id)\n", childName)
   fmt.Printf("   if (dummy.Lastupdate != childObject.Lastupdate) {\n")
   fmt.Printf("      return true\n")
   fmt.Printf("   }\n")
   fmt.Printf("   return false\n")
   fmt.Printf("}\n\n")
}

func genLoadObjects(tablename string) {

   var fieldsComma = ""    // id, name, ipadres, description"
   var fieldsAnd = ""      // &id, &name, &ipadres, &description"
   var fieldsAssign = ""   // Id:id, Name:name, Ipadres:ipadres, Description:description"
   var fieldsVarList = ""  // var id int\n      var ipadres string\n      var name string\n      var description string\n"

   // for each field
   queryTables := fmt.Sprintf("SELECT COLUMN_NAME, DATA_TYPE FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = '%s' AND TABLE_NAME = '%s';", conf.DBname, tablename)
   rowFields, err := mydb.DB.Query(queryTables)
   for rowFields.Next() {
      var FieldName string
      var DATA_TYPE string
      err = rowFields.Scan(&FieldName, &DATA_TYPE)
      logerror.CheckErr(err)
      fieldsComma = fmt.Sprintf("%s%s, ", fieldsComma, sqlfieldname(FieldName))
      fieldsAnd = fmt.Sprintf("%s&%s, ", fieldsAnd, dbfieldname(FieldName))
      fieldsAssign = fmt.Sprintf("%s%s:%s, ", fieldsAssign, dbfieldname(FieldName), dbfieldname(FieldName))
      fieldsVarList = fmt.Sprintf("%s      var %s %s\n", fieldsVarList, dbfieldname(FieldName), DataTypeConversion(DATA_TYPE))
   }

   // trim comma's
   fieldsComma = fmt.Sprintf("%s", TrimSuffix(fieldsComma,", "))
   fieldsAnd = fmt.Sprintf("%s", TrimSuffix(fieldsAnd,", "))
   fieldsAssign = fmt.Sprintf("%s", TrimSuffix(fieldsAssign,", "))

   // object naming
   var object = createSingleObjectName(tablename)
   var objects = createCollectionName(tablename)

  fmt.Printf("func Load%s(dummy *[]%s, filter string) {\n", objects, object)
        fmt.Printf("   query := fmt.Sprintf(\"SELECT %s FROM %s %s ORDER BY name\", filter)\n", fieldsComma, tablename, ws)
        fmt.Printf("   rows, err := mydb.DB.Query(query)\n")
        fmt.Printf("   logerror.CheckErr(err)\n")
        fmt.Printf("   defer rows.Close()\n")
        fmt.Printf("   for rows.Next() {\n")
        // var list
        fmt.Printf("%s", fieldsVarList)
        fmt.Printf("      err = rows.Scan(%s)\n", fieldsAnd)
        fmt.Printf("      logerror.CheckErr(err)\n")
        fmt.Printf("      *dummy = append(*dummy, %s{%s})\n", object, fieldsAssign)
        fmt.Printf("   }\n")
        fmt.Printf("}\n\n")
}

func genSave(tablename string) {

   var object = createSingleObjectName(tablename)

   var fieldsComma = ""    // id, name, ipadres, description"
   var fieldsValues = ""
   var fieldNames = ""
   var fieldQuestion = ""

   // for each field
   queryTables := fmt.Sprintf("SELECT COLUMN_NAME, DATA_TYPE FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = '%s' AND TABLE_NAME = '%s';", conf.DBname, tablename)
   rowFields, err := mydb.DB.Query(queryTables)
   for rowFields.Next() {
      var FieldName string
      var DATA_TYPE string
      err = rowFields.Scan(&FieldName, &DATA_TYPE)
      logerror.CheckErr(err)

      _, ok := reservedWords[FieldName] //reserved word check
      if (!ok) {
         fieldsComma = fmt.Sprintf("%s%s, ", fieldsComma, sqlfieldname(FieldName))
         fieldQuestion = fmt.Sprintf("%s?, ", fieldQuestion)

         switch DATA_TYPE {
           case "mediumtext":
              fieldsValues = fmt.Sprintf("%s%s, ", fieldsValues, "'%s'")
           case "varchar":
              fieldsValues = fmt.Sprintf("%s%s, ", fieldsValues, "'%s'")
           case "int":
              fieldsValues = fmt.Sprintf("%s%s, ", fieldsValues, "%d")
           default:
              panic(fmt.Sprintf("unimplemented datatype: %s\n", DATA_TYPE))
         }

         fieldNames  = fmt.Sprintf("%s childObject.%s, ", fieldNames, FirstLetterUpcase(FieldName))
      }
   }

   // trim comma's
   fieldsComma  = fmt.Sprintf("%s", TrimSuffix(fieldsComma, ", "))
   fieldsValues = fmt.Sprintf("%s", TrimSuffix(fieldsValues,", "))
   fieldNames  = fmt.Sprintf("%s", TrimSuffix(fieldNames, ", "))
   fieldQuestion = fmt.Sprintf("%s", TrimSuffix(fieldQuestion, ", "))

   fmt.Printf("func (childObject *%s) Save() {\n", object)
   fmt.Printf("   strQuery := fmt.Sprintf(\"INSERT INTO %s (%s) VALUES (%s)\")\n", tablename, fieldsComma, fieldQuestion)
   fmt.Printf("   instcmd, err := mydb.DB.Prepare (strQuery)\n")
   fmt.Printf("   logerror.CheckErr(err)\n")
   fmt.Printf("   defer instcmd.Close()\n")
   fmt.Printf("   instcmd.Exec(%s)\n", fieldNames)
   fmt.Printf("}\n\n")
}

func genUpdate(tablename string) {

   var object = createSingleObjectName(tablename)

   var fieldsComma = ""    // id, name, ipadres, description"
   var fieldsValues = ""
   var fieldNames = ""

   // for each field
   queryTables := fmt.Sprintf("SELECT COLUMN_NAME, DATA_TYPE FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = '%s' AND TABLE_NAME = '%s';", conf.DBname, tablename)
   rowFields, err := mydb.DB.Query(queryTables)
   for rowFields.Next() {
      var FieldName string
      var DATA_TYPE string
      err = rowFields.Scan(&FieldName, &DATA_TYPE)
      logerror.CheckErr(err)

      _, ok := reservedWords[FieldName] //reserved word check
      if (!ok) {
         fieldsComma = fmt.Sprintf("%schildObject.%s, ", fieldsComma, FieldName)

         switch DATA_TYPE {
           case "mediumtext":
              fieldsValues = fmt.Sprintf("%s%s=%s, ", fieldsValues, FieldName, "?")
           case "varchar":
              fieldsValues = fmt.Sprintf("%s%s=%s, ", fieldsValues, FieldName, "?")
           case "int":
              fieldsValues = fmt.Sprintf("%s%s=%s, ", fieldsValues, FieldName, "?")
           default:
              panic(fmt.Sprintf("unimplemented datatype: %s\n", DATA_TYPE))
         }

         fieldNames  = fmt.Sprintf("%s childObject.%s, ", fieldNames, FirstLetterUpcase(FieldName))
      }
   }

   // trim comma's
   fieldsComma  = fmt.Sprintf("%s", TrimSuffix(fieldsComma, ", "))
   fieldsValues = fmt.Sprintf("%s", TrimSuffix(fieldsValues,", "))
   fieldNames  = fmt.Sprintf("%s", TrimSuffix(fieldNames, ", "))

   fmt.Printf("func (childObject *%s) Update() {\n", object)
   // check collision (new)
   fmt.Printf("   if (childObject.Checkcollision()) {\n")
   fmt.Printf("      fmt.Printf(\"Collision on %s (id=%%d)\\n\", childObject.Id)\n", object)
   fmt.Printf("   }\n")

   fmt.Printf("   strQuery := fmt.Sprintf(\"UPDATE %s SET %s, lastupdate=now() WHERE id=?;\") \n", tablename, fieldsValues)
   fmt.Printf("   instcmd, err := mydb.DB.Prepare (strQuery)\n")
   fmt.Printf("   logerror.CheckErr(err)\n")
   fmt.Printf("   defer instcmd.Close()\n")
   fmt.Printf("   instcmd.Exec(%s,childObject.Id)\n", fieldNames)
   fmt.Printf("}\n\n")
}

func genDisable(tablename string) {

   var object = createSingleObjectName(tablename)

   fmt.Printf("func (childObject *%s) Disable() {\n", object)
   fmt.Printf("   strQuery := fmt.Sprintf(\"UPDATE %s SET enabled=0, lastupdate=now() WHERE id=%s;\", %s) \n", tablename, "%d", "childObject.Id")
   fmt.Printf("   instcmd, err := mydb.DB.Prepare (strQuery)\n")
   fmt.Printf("   logerror.CheckErr(err)\n")
   fmt.Printf("   defer instcmd.Close()\n")
   fmt.Printf("   instcmd.Exec()\n")
   fmt.Printf("}\n\n")
}

func genEnable(tablename string) {

   var object = createSingleObjectName(tablename)

   fmt.Printf("func (childObject *%s) Enable() {\n", object)
   fmt.Printf("   strQuery := fmt.Sprintf(\"UPDATE %s SET enabled=1, lastupdate=now() WHERE id=%s;\", %s) \n", tablename, "%d", "childObject.Id")
   fmt.Printf("   instcmd, err := mydb.DB.Prepare (strQuery)\n")
   fmt.Printf("   logerror.CheckErr(err)\n")
   fmt.Printf("   defer instcmd.Close()\n")
   fmt.Printf("   instcmd.Exec()\n")
   fmt.Printf("}\n\n")
}

func genStruct() {
   // get tables
   queryTables := fmt.Sprintf("SHOW TABLES")
   rowTables, err := mydb.DB.Query(queryTables)
   logerror.CheckErr(err)

   // for each table
   for rowTables.Next() {
      var tablename string
      err = rowTables.Scan(&tablename)
      logerror.CheckErr(err)
      genStructTable(tablename)
   }
}

func genStructTable (tablename string) {
   fmt.Printf("type %s struct {\n", createSingleObjectName(tablename))

   // for each field
   queryTables := fmt.Sprintf("SELECT COLUMN_NAME, DATA_TYPE FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = '%s' AND TABLE_NAME = '%s';", conf.DBname, tablename)
   rowFields, err := mydb.DB.Query(queryTables)
   for rowFields.Next() {
      var FieldName string
      var DATA_TYPE string
      err = rowFields.Scan(&FieldName, &DATA_TYPE)
      logerror.CheckErr(err)
      fmt.Printf("   %s %s \t`json:\"%s\"` \n", dbfieldname(FieldName), DataTypeConversion(DATA_TYPE), strings.ToLower(dbfieldname(FieldName)))
   }

   fmt.Printf("}\n\n")
}

func DataTypeConversion (datatype string) string {

   switch datatype{
                case "int":             return "int"
                case "varchar":         return "string"
                case "mediumtext":      return "string"
                case "bigint":          return "int64"
                case "smallint":        return "int"
                case "timestamp":       return "time.Time"
                case "datetime":        return "string"
                case "text":            return "string"
   }

   return "geen idee:" + datatype
}

func sqlfieldname(field string) string {

   field = strings.ToLower(field)

   if (field=="switch") {field = "switch as myswitch"}
   if (field=="interface") {field = "interface as myinterface"}
   if _, err := strconv.Atoi(string([]rune(field)[0])); err == nil {
        //numeric
        field="my" + field
   }
        return field
}

func dbfieldname(field string) string {
   field = strings.ToLower(field)

   if (field=="switch") {field = "myswitch"}
   if (field=="interface") {field = "myinterface"}
   if _, err := strconv.Atoi(string([]rune(field)[0])); err == nil {
        //numeric
        field="my" + field
   }

   field = FirstLetterUpcase(field)
   return field
}

func createSingleObjectName(tablename string) string {

   tablename = strings.ToLower(tablename)

   // should be warnings for these types:
   if (tablename=="switch") {tablename = "myswitch"}
   if (tablename=="interface") {tablename = "myinterfaces"}
   if _, err := strconv.Atoi(string([]rune(tablename)[0])); err == nil {
        //numeric
        tablename="my" + tablename
   }

   tablename = TrimSuffix(tablename,"s")

   for i, v := range tablename {
      return string(unicode.ToUpper(v)) + tablename[i+1:]
    }
    return ""
}

func createCollectionName(tablename string) string {

        return createSingleObjectName(tablename) + "s"
}

func TrimSuffix(s, suffix string) string {
    if strings.HasSuffix(s, suffix) {
        s = s[:len(s)-len(suffix)]
    }
    return s
}

func FirstLetterUpcase(s string) string {
   for i, v := range s {
      return string(unicode.ToUpper(v)) + s[i+1:]
   }
   return ""
}

func init() {
   // load settings & open database
   configuration.Loadsettings (&conf)
   mydb.InitConfigSettings(conf)
}

func close() {
   // close db
   mydb.Close()
}

