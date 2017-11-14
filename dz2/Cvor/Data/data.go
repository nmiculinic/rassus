package Data

import (
	"encoding/csv"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

func ReadCSV(csvFile string) (records [][]string, err error) {
	f, err := os.Open(csvFile)
	defer f.Close()
	if err != nil {
		return nil, err
	}
	r := csv.NewReader(f)
	records, err = r.ReadAll()
	return
}
func ReadMeasurement(startTime time.Time, rec [][]string) (float64, error) {
	elapsedSeconds := time.Now().Sub(startTime).Seconds()
	no := (int(elapsedSeconds) % 100) + 1
	log.Printf("Elapsed seconds %f, field %d, data %s\n", elapsedSeconds, no, rec[no])
	if rec[no][3] != "" {
		if val, err := strconv.ParseFloat(rec[no][3], 64); err != nil {
			log.Println(
				fmt.Sprintf(
					"Missing value for Row %d, param %s ",
					no,
					"CO"), err)
			return 0.0, err
		} else {
			return val, nil
		}
	} else {
		log.Println(
			fmt.Sprintf(
				"Missing value for Row %d, param %s ",
				no,
				"CO"))
		return 0.0, errors.New("Missing value, empty str")
	}
}
