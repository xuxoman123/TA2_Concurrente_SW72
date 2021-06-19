package main

//para correr:
//go mod init github.com/godie910/restapi
//go mod tidy
//go build github.com/godie910/restapi
// ./restapi

import (
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"

	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

//inputs entrada
type JSON_Input struct {
	X float64 `json:"x"` // coordenada x(x, )
	Y float64 `json:"y"` // coordenada y( ,y)
	K []byte  `json:"k"` // array de vecinos (k)
}

type JSON_Output struct {
	Data    []Data     `json:"data"`
	Paths   [][]Labels `json:"paths"`
	Classes []string   `json:"classes"`
}




var retorno JSON_Output

//datos de la etiqueta cruzada (x,y,vecino)
type Punto struct {
	X     float64 `json:"x"`     // coordenada x(x, )
	Y     float64 `json:"y"`     // coordenada y( ,y)
	Label string  `json:"label"` // tipo (vecino)
}


type Labels struct {
	Nombre string `json:"nombre"` //nombre del tipo
	Cont   int    `json:"cont"`   //contador de tipos (#vecinos)
}

func (p Punto) String() string {
	return fmt.Sprintf("[*] X = %f, Y = %f Label = %s\n", p.X, p.Y, p.Label) //punto intersección (etiqueta)
}

type Extras struct {
	Extra1 float64 `json:"extra1"` //extra1
	Extra2 string  `json:"extra2"` //extra2
}

type Data struct {
	Punto     Punto   `json:"punto"`     // punto de intersección (x,y,tipo)
	Distancia float64 `json:"distancia"` // distancia euclidiana del vecino hasta el punto
	Tipo   string `json:"tipo"`   //tipo de instrumento
	Estado  string `json:"estado"`  //estado del estudio ambiental
}

func (d Data) String() string {
	return fmt.Sprintf(
		"X = %f Y = %f, Distance = %f Label = %s, Tipo = %s, Estado = %s\n",
		d.Punto.X, d.Punto.Y, d.Distancia, d.Punto.Label, d.Tipo, d.Estado,   //valores de data incluyendo distancia
	)
}



type Block []Data

func (b Block) Len() int           { return len(b) }
func (b Block) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b Block) Less(i, j int) bool { return b[i].Distancia < b[j].Distancia }


func DEuclidiana(A Punto, X Punto) (distancia float64, err error) {

	distancia = math.Sqrt(math.Pow((X.X-A.X), 2) + math.Pow((X.Y-A.Y), 2))      //formula euclidiana para hallar la distancia 
	if distancia < 0 {
		return 0, fmt.Errorf("ERROR: Distancia euclidiana erronea, negativa") //error de datos para la distancia
	}
	return distancia, nil
}

func LoadData(url string) (data []Data, err error) {
	// abrimos
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	reader := csv.NewReader(resp.Body)
	// leemos y lo separamos por coma
	reader.Comma = ','

	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}






	fmt.Println("[*] Cargando...")
	fmt.Println()
	filas := len(records)
	columnas := len(records[0])
	if columnas < 3 {
		return nil, fmt.Errorf("no se puede cargar esta data")
	}
	for i := 0; i < filas; i++ {
		for j := 0; j < columnas; j++ {
			fmt.Printf("%s\t  ", records[i][j])
		}
		if i == 0 {
			fmt.Println()
		}
		fmt.Println()
	}
	fmt.Println()
	var value float64
	data = make([]Data, filas-1, filas-1)
	for i := 1; i < filas; i++ {
		value, err = strconv.ParseFloat(records[i][0], 64)
		if err != nil {
			return nil, fmt.Errorf("no se puede analizar el valor X: %v", err)
		}
		data[i-1].Punto.X = value
		value, err = strconv.ParseFloat(records[i][1], 64)
		if err != nil {
			return nil, fmt.Errorf("no se puede analizar el valor Y: %v", err)
		}
		data[i-1].Punto.Y = value
		data[i-1].Punto.Label = records[i][2]

		data[i-1].Tipo = records[i][3]
		data[i-1].Estado = records[i][4]



	}
	return data, nil
}

func ValidError(err error) {   //validación
	if err != nil {
		fmt.Printf("[!] %s\n", err.Error())
		os.Exit(1)
	}
}

func Knn(data []Data, k byte, X *Punto) (err error) {      //algoritmo Knn (K vecinos más próximos)
	n := len(data)
	// Cálculo de la distancia entre los puntos y X
	for i := 0; i < n; i++ {
		if data[i].Distancia, err = DEuclidiana(data[i].Punto, *X); err != nil {
			return err
		}
	}

	var blk Block 
	blk = data
	// Ordena la data de menor a mayor
	sort.Sort(blk)
	var save []Labels
	if int(k) > n {
		return nil
	}
	for i := byte(0); i < k; i++ {
		save = IncrementoLabels(data[i].Punto.Label, save) //pasa
	}

	fmt.Printf("[*] cantidad de vecinos(k) = %d\n", k)  //print #vecinos utilizado
	fmt.Println()
	fmt.Printf("[*] %+v\n", save)
	fmt.Println()

	retorno.Paths = append(retorno.Paths, save)
	
	max := 0
	var maxLabel string              //etiquetas
	m := len(save)
	for i := 0; i < m; i++ {
		if max < save[i].Cont {
			max = save[i].Cont
			maxLabel = save[i].Nombre
		}
	}

	X.Label = maxLabel
	retorno.Classes = append(retorno.Classes, maxLabel)
	return nil
}

func IncrementoLabels(label string, labels []Labels) []Labels {   //etiquetas
	if labels == nil {
		labels = append(labels, Labels{
			Nombre: label,
			Cont:   1,
		})
		return labels
	}

	cont := len(labels)
	for i := 0; i < cont; i++ {
		if strings.Compare(labels[i].Nombre, label) == 0 {
			labels[i].Cont++
			return labels
		}
	}

	return append(labels, Labels{
		Nombre: label,
		Cont:   1,
	})
}

func API_KNN(w http.ResponseWriter, r *http.Request) {          //API del algoritmo
	w.Header().Set("Counter-Type", "application/json")

	//fmt.Println()

	//data, err := LoadData("Reporte_Proyecto_APROBADO.csv")     //data que se correrá
	//ValidError(err)

	url := "https://raw.githubusercontent.com/xuxoman123/data/main/Reporte_Proyecto_APROBADO.csv"
	data, err := LoadData(url)
	ValidError(err)
	if err != nil {
		panic(err)
		
	}

	// read from JSON
	var json_input JSON_Input         //entrada inputs json
	_ = json.NewDecoder(r.Body).Decode(&json_input)
	var X Punto
	X.X = json_input.X
	X.Y = json_input.Y
	var k = json_input.K

	n := len(k)
	for i := 0; i < n; i++ {
		err = Knn(data, k[i], &X)
		if i == 0 {
			fmt.Println(data)
			retorno.Data = data
		}
		
		ValidError(err)
		fmt.Printf("[*] Result for X is ")
		fmt.Println(X)
	}

	json.NewEncoder(w).Encode(retorno)
	var aux JSON_Output
    retorno = aux
}

func main() {

	
	//Init router
	r := mux.NewRouter()

	//Route Handlers / Endpoints
	r.HandleFunc("/postman/KnnConcu", API_KNN).Methods("POST")         //metodos del API

	//log.Fatal(http.ListenAndServe(":5000", r))
	log.Fatal(
		http.ListenAndServe(
			":5000",
			handlers.CORS(
				handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization"}),
				handlers.AllowedMethods([]string{"GET", "POST", "PUT", "HEAD", "OPTIONS"}),
				handlers.AllowedOrigins([]string{"*"}))(r)))
}
