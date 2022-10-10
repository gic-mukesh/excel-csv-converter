package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"

	excelize "github.com/xuri/excelize/v2"
)

const maxUploadSize = 10 * 1024 * 1024 // 10 mb
const dir = "data/download/"

var fileName string

func convertExcelToJson(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if r.Method != "POST" {
		respondWithError(w, http.StatusBadRequest, "Invalid Method")
		return
	}

	err := r.ParseMultipartForm(32 << 20)
	checkError(w, err)

	files := r.MultipartForm.File["file"]

	if len(files) != 1 {
		respondWithError(w, http.StatusBadRequest, "Process One excel file only!")
		return
	}

	for _, fileHeader := range files {
		fileName = fileHeader.Filename

		segments := strings.Split(fileName, ".")
		extension := segments[len(segments)-1]
		if extension != "xlsx" {
			respondWithError(w, http.StatusBadRequest, "Process having xlsx extension only!")
			return
		}
		if fileHeader.Size > maxUploadSize {
			respondWithError(w, http.StatusBadRequest, "Process file less than 1MB in size!")
			return
		}

		// Open the file
		file, err := fileHeader.Open()
		checkError(w, err)

		defer file.Close()

		buff := make([]byte, 512)
		_, err = file.Read(buff)
		checkError(w, err)

		_, err = file.Seek(0, io.SeekStart)
		checkError(w, err)

		err = os.MkdirAll(dir, os.ModePerm)
		checkError(w, err)

		f, err := os.Create(dir + fileHeader.Filename)
		checkError(w, err)

		defer f.Close()

		_, err = io.Copy(f, file)
		checkError(w, err)
	}
	excelToJson(w, fileName)
}

func excelToJson(w http.ResponseWriter, file string) {
	var (
		headers   []string
		result    []*map[string]interface{}
		wb        = new(excelize.File)
		err       error
		sheetName string
	)
	wb, err = excelize.OpenFile(dir + file)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error while opening file")
		return
	}

	xlsx, err := excelize.OpenFile(dir + fileName)
	if err != nil {
		log.Println(err)
		respondWithError(w, http.StatusBadRequest, "Error while getting sheet name!")
	}
	i := 0
	for _, sheet := range xlsx.GetSheetMap() {
		if i == 0 {
			sheetName = sheet
		}
		i++
	}
	rows, _ := wb.GetRows(sheetName)
	headers = rows[0]
	for _, row := range rows[1:] {
		var tmpMap = make(map[string]interface{})
		for j, v := range row {
			tmpMap[strings.Join(strings.Split(headers[j], " "), "")] = v
		}
		result = append(result, &tmpMap)
	}
	respondWithJson(w, http.StatusAccepted, result)
}

func convertCsvToJson(w http.ResponseWriter, r *http.Request) {

	err := r.ParseMultipartForm(32 << 20)
	checkError(w, err)

	files := r.MultipartForm.File["file"][0]
	fileName = files.Filename

	segments := strings.Split(fileName, ".")
	extension := segments[len(segments)-1]
	if extension != "csv" {
		respondWithError(w, http.StatusBadRequest, "Please provide excel file having csv extension")
		return
	}
	// data, err := readCsv(files)
	fileBytes := readCsv(files)
	respondWithJson(w, http.StatusAccepted, fileBytes)

	// if err != nil {
	// 	panic(fmt.Sprintf("error while handling csv file: %s\n", err))
	// }

	// json, err := csvToJson(data)
	// if err != nil {
	// 	panic(fmt.Sprintf("error while converting csv to json file: %s\n", err))
	// }

	// respondWithJson(w, http.StatusAccepted, json)
}

// func readCsv(file *multipart.FileHeader) ([][]string, error) {
// 	csvFile, err := file.Open()
// 	var rows [][]string

// 	if err != nil {
// 		log.Fatal("File not found in the given directory!")
// 	}

// 	reader := csv.NewReader(csvFile)
// 	content, _ := reader.ReadAll()

// 	if len(content) < 1 {
// 		log.Fatal("The file maybe empty or length of the lines are not the same")
// 	}

// 	for {
// 		row, err := reader.Read()
// 		if err == io.EOF {
// 			break
// 		}

// 		if err != nil {
// 			return rows, fmt.Errorf("failed to parse csv: %s", err)
// 		}

// 		rows = append(rows, row)
// 	}

// 	// return rows, nil

// 	// headersArr := make([]string, 0)
// 	// for _, headE := range content[0] {
// 	// 	headersArr = append(headersArr, headE)
// 	// }

// 	// //Remove the header row
// 	// content = content[1:]

// 	// var buffer bytes.Buffer
// 	// buffer.WriteString("[")
// 	// for i, d := range content {
// 	// 	buffer.WriteString("{")
// 	// 	for j, y := range d {
// 	// 		buffer.WriteString(`"` + headersArr[j] + `":`)
// 	// 		buffer.WriteString((`"` + y + `"`))
// 	// 		//end of property
// 	// 		if j < len(d)-1 {
// 	// 			buffer.WriteString(",")
// 	// 		}

// 	// 	}
// 	// 	//end of object of the array
// 	// 	buffer.WriteString("}")
// 	// 	if i < len(content)-1 {
// 	// 		buffer.WriteString(",")
// 	// 	}
// 	// }

// 	// buffer.WriteString(`]`)
// 	// //rawMessage := json.RawMessage(buffer.String())
// 	// //x, err := json.MarshalIndent(rawMessage, "", "  ")
// 	// f := buffer.String()
// 	// f = strings.ReplaceAll(f, `\`, "")
// 	// fmt.Println(f)
// 	return rows, nil
// }

func readCsv(file *multipart.FileHeader) string {
	csvFile, err := file.Open()

	if err != nil {
		log.Fatal("The file is not found || wrong root")
	}
	defer csvFile.Close()

	reader := csv.NewReader(csvFile)
	content, _ := reader.ReadAll()

	if len(content) < 1 {
		log.Fatal("Something wrong, the file maybe empty or length of the lines are not the same")
	}

	headersArr := make([]string, 0)
	for _, headE := range content[0] {
		headersArr = append(headersArr, headE)
	}

	//Remove the header row
	content = content[1:]

	var buffer bytes.Buffer
	buffer.WriteString("[")
	for i, d := range content {
		buffer.WriteString("{")
		for j, y := range d {
			buffer.WriteString(`"` + headersArr[j] + `":`)
			buffer.WriteString((`"` + y + `"`))
			//end of property
			if j < len(d)-1 {
				buffer.WriteString(",")
			}

		}
		//end of object of the array
		buffer.WriteString("}")
		if i < len(content)-1 {
			buffer.WriteString(",")
		}
	}

	buffer.WriteString(`]`)
	//rawMessage := json.RawMessage(buffer.String())
	//x, err := json.MarshalIndent(rawMessage, "", "  ")
	f := buffer.String()
	f = strings.ReplaceAll(f, `\`, "")
	fmt.Println(f)
	return f
}

func csvToJson(rows [][]string) (string, error) {
	var entries []map[string]interface{}
	attributes := rows[0]
	for _, row := range rows[1:] {
		entry := map[string]interface{}{}
		for i, value := range row {
			attribute := attributes[i]
			// split csv header key for nested objects
			objectSlice := strings.Split(attribute, ".")
			internal := entry
			for index, val := range objectSlice {
				// split csv header key for array objects
				key, arrayIndex := arrayContentMatch(val)
				if arrayIndex != -1 {
					if internal[key] == nil {
						internal[key] = []interface{}{}
					}
					internalArray := internal[key].([]interface{})
					if index == len(objectSlice)-1 {
						internalArray = append(internalArray, value)
						internal[key] = internalArray
						break
					}
					if arrayIndex >= len(internalArray) {
						internalArray = append(internalArray, map[string]interface{}{})
					}
					internal[key] = internalArray
					internal = internalArray[arrayIndex].(map[string]interface{})
				} else {
					if index == len(objectSlice)-1 {
						internal[key] = value
						break
					}
					if internal[key] == nil {
						internal[key] = map[string]interface{}{}
					}
					internal = internal[key].(map[string]interface{})
				}
			}
		}
		entries = append(entries, entry)
	}

	bytes, err := json.MarshalIndent(entries, "", "	")
	if err != nil {
		return "", fmt.Errorf("Marshal error %s\n", err)
	}

	return string(bytes), nil
}

func arrayContentMatch(str string) (string, int) {
	i := strings.Index(str, "[")
	if i >= 0 {
		j := strings.Index(str, "]")
		if j >= 0 {
			index, _ := strconv.Atoi(str[i+1 : j])
			return str[0:i], index
		}
	}
	return str, -1
}

func checkError(w http.ResponseWriter, err error) {
	if err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("%v", err))
		return
	}
}

func respondWithJson(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	respondWithJson(w, code, map[string]string{"error": msg})
}

func main() {
	http.HandleFunc("/convert-excel-to-json", convertExcelToJson)
	http.HandleFunc("/convert-csv-to-json", convertCsvToJson)
	log.Println("Service Started at 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
