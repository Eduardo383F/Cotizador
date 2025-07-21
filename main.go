package main

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/phpdave11/gofpdf"
)

type Producto struct {
	Cantidad       int
	Parte          string
	Descripcion    string
	PrecioUnitario float64
	PrecioTotal    float64
}

type VistaPrevia struct {
	Fecha        string
	Destinatario string
	Concepto     string
	IVA          bool
	Productos    []Producto
}

func main() {
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("templates"))))
	http.HandleFunc("/", mostrarFormulario)
	http.HandleFunc("/vista-previa", vistaPrevia)
	http.HandleFunc("/descargar-pdf", descargarPDF)

	fmt.Println("Servidor corriendo en http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}

func mostrarFormulario(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("templates/form.html"))
	tmpl.Execute(w, nil)
}

func vistaPrevia(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	iva := r.FormValue("iva") == "on"

	fecha := r.FormValue("fecha")
	destinatario := r.FormValue("destinatario")
	concepto := r.FormValue("concepto")

	// Leer productos
	var productos []Producto
	cantidades := r.Form["cantidad"]
	partes := r.Form["parte"]
	descripciones := r.Form["descripcion"]
	precios := r.Form["precio"]

	for i := range cantidades {
		cant, _ := strconv.Atoi(cantidades[i])
		precio, _ := strconv.ParseFloat(precios[i], 64)

		if iva {
			precio *= 1.16
			precio = float64(int(precio)) // quitar decimales
		}

		prod := Producto{
			Cantidad:       cant,
			Parte:          partes[i],
			Descripcion:    descripciones[i],
			PrecioUnitario: precio,
			PrecioTotal:    float64(cant) * precio,
		}
		productos = append(productos, prod)
	}

	data := VistaPrevia{
		Fecha:        strings.ToUpper(fecha),
		Destinatario: strings.ToUpper(destinatario),
		Concepto:     strings.ToUpper(concepto),
		IVA:          iva,
		Productos:    productos,
	}

	tmpl := template.Must(template.ParseFiles("templates/preview.html"))
	tmpl.Execute(w, data)
}

func descargarPDF(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	iva := r.FormValue("iva") == "true" || r.FormValue("iva") == "on"

	fecha := strings.ToUpper(r.FormValue("fecha"))
	destinatario := strings.ToUpper(r.FormValue("destinatario"))
	concepto := strings.ToUpper(r.FormValue("concepto"))

	// Leer productos
	var productos []Producto
	cantidades := r.Form["cantidad"]
	partes := r.Form["parte"]
	descripciones := r.Form["descripcion"]
	precios := r.Form["precio"]

	for i := range cantidades {
		cant, _ := strconv.Atoi(cantidades[i])
		precio, _ := strconv.ParseFloat(precios[i], 64)
		if iva {
			precio *= 1.16
			precio = float64(int(precio))
		}
		productos = append(productos, Producto{
			Cantidad:       cant,
			Parte:          partes[i],
			Descripcion:    descripciones[i],
			PrecioUnitario: precio,
			PrecioTotal:    float64(cant) * precio,
		})
	}

	// Crear PDF
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddUTF8Font("DejaVu", "", "DejaVuSans.ttf")
	pdf.SetFont("DejaVu", "", 11)
	pdf.AddPage()

	// Logo
	pdf.Image("logo.png", 10, 10, 40, 0, false, "", 0, "")

	// Pie de página con datos fiscales

	pdf.MultiCell(150, 15, `
FELIPE FERNANDEZ LOPEZ
R.F.C FELF740411830
DIRECCIÓN: CALLE 20 DE NOVIEMBRE S/N, COL. BENITO JUAREZ, MIXQUIAHUALA DE JUAREZ, HGO. C.P 42719
`, "", "L", false)

	// Fecha (esquina superior derecha)
	pdf.SetXY(150, 15)
	pdf.CellFormat(0, 10, fecha, "", 0, "R", false, 0, "")

	pdf.Ln(20)
	pdf.CellFormat(0, 10, "AT'N: "+destinatario, "", 1, "", false, 0, "")
	pdf.CellFormat(0, 10, "COTIZACIÓN: "+concepto, "", 1, "", false, 0, "")

	pdf.Ln(5)
	// Tabla
	pdf.SetFillColor(200, 200, 200)
	pdf.CellFormat(15, 10, "CANT.", "1", 0, "C", true, 0, "")
	pdf.CellFormat(35, 10, "NO. DE PARTE", "1", 0, "C", true, 0, "")
	pdf.CellFormat(70, 10, "DESCRIPCIÓN", "1", 0, "C", true, 0, "")
	pdf.CellFormat(30, 10, "P. UNITARIO", "1", 0, "C", true, 0, "")
	pdf.CellFormat(30, 10, "TOTAL", "1", 1, "C", true, 0, "")

	for _, p := range productos {
		pdf.CellFormat(15, 10, fmt.Sprint(p.Cantidad), "1", 0, "C", false, 0, "")
		pdf.CellFormat(35, 10, p.Parte, "1", 0, "", false, 0, "")
		pdf.CellFormat(70, 10, p.Descripcion, "1", 0, "", false, 0, "")
		pdf.CellFormat(30, 10, fmt.Sprintf("$%.0f", p.PrecioUnitario), "1", 0, "R", false, 0, "")
		pdf.CellFormat(30, 10, fmt.Sprintf("$%.0f", p.PrecioTotal), "1", 1, "R", false, 0, "")
	}

	pdf.Ln(10)
	// Pie de página con datos fiscales
	pdf.MultiCell(0, 8, `
1. Precios netos con IVA en pesos mexicanos (MXN)
2. Es necesario cubrir el importe total de las piezas al contado.
3. Vigencia de cotización: 30 días después de su emisión.
4. Flete incluido al lugar de preferencia del cliente.
`, "", "L", false)

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=cotizacion.pdf")
	_ = pdf.Output(w)
}
