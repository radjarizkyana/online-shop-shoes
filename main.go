package main

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"text/template"
	"sort"
	"strings"
)

type Akun struct {
	Username  string
	Password  string
	Tipe      string // "admin", "pemilik", "pembeli"
	Disetujui bool
}

type Barang struct {
	Nama      string
	Harga     int
	Kuantitas int
}

type Transaksi struct {
	UsernamePembeli string
	BarangDibeli    Barang
	Jumlah          int
}

var (
	akunList       []Akun
	barangList     []Barang
	transaksiList  []Transaksi
	templatesCache = make(map[string]*template.Template)
)

func init() {
	loadTemplates()
}

type OwnerData struct {
	BarangList     []Barang
	TransaksiList  []Transaksi
}

type BuyerData struct {
	Username   string
	BarangList []Barang
}

func loadTemplates() {
	templates := []string{"index", "register", "login", "admin", "ownerr", "buyer", "transactions"}
	for _, tmpl := range templates {
		t, err := template.ParseFiles("templates/" + tmpl + ".html")
		if err != nil {
			log.Fatal("Error loading template:", tmpl, err)
		}
		templatesCache[tmpl] = t
	}
}

func clearScreen() {
	switch runtime.GOOS {
	case "linux", "darwin":
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		cmd.Run()
	case "windows":
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	default:
		fmt.Println("Platform tidak didukung.")
	}
}

func waitForEnter() {
	fmt.Println("Tekan Enter untuk melanjutkan...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}

func saveDataToFile(filename, data string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(data)
	if err != nil {
		return err
	}

	return nil
}

func formatDataToText() string {
	formattedData := "Akun List:\n"
	for _, akun := range akunList {
		formattedData += fmt.Sprintf("Username: %s, Password: %s, Tipe: %s, Disetujui: %t\n", akun.Username, akun.Password, akun.Tipe, akun.Disetujui)
	}

	formattedData += "\nBarang List:\n"
	for _, barang := range barangList {
		formattedData += fmt.Sprintf("Nama: %s, Harga: %d, Kuantitas: %d\n", barang.Nama, barang.Harga, barang.Kuantitas)
	}

	formattedData += "\nTransaksi List:\n"
	for _, transaksi := range transaksiList {
		formattedData += fmt.Sprintf("Pembeli: %s, Barang: %s, Jumlah: %d\n", transaksi.UsernamePembeli, transaksi.BarangDibeli.Nama, transaksi.Jumlah)
	}

	return formattedData
}

func saveData() {
	file, err := os.Create("data.gob")
	if err != nil {
		log.Println("Error saving data:", err)
		return
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)
	err = encoder.Encode(akunList)
	if err != nil {
		log.Println("Error encoding akunList:", err)
		return
	}
	err = encoder.Encode(barangList)
	if err != nil {
		log.Println("Error encoding barangList:", err)
		return
	}
	err = encoder.Encode(transaksiList)
	if err != nil {
		log.Println("Error encoding transaksiList:", err)
		return
	}

	log.Println("Data berhasil disimpan.")

	dataText := formatDataToText()
	err = saveDataToFile("data.txt", dataText)
	if err != nil {
		log.Println("Error saving data to text file:", err)
		return
	}
	log.Println("Data berhasil disimpan dalam format teks.")
}

func loadData() {
	file, err := os.Open("data.gob")
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("File data tidak ditemukan, memulai dengan data kosong.")
			return
		}
		log.Println("Error loading data:", err)
		return
	}
	defer file.Close()

	decoder := gob.NewDecoder(file)
	err = decoder.Decode(&akunList)
	if err != nil {
		log.Println("Error decoding akunList:", err)
		return
	}
	err = decoder.Decode(&barangList)
	if err != nil {
		log.Println("Error decoding barangList:", err)
		return
	}
	err = decoder.Decode(&transaksiList)
	if err != nil {
		log.Println("Error decoding transaksiList:", err)
		return
	}
	log.Println("Data berhasil dimuat.")
}

func main() {
	admin := Akun{Username: "admin", Password: "admin123", Tipe: "admin", Disetujui: true}
	akunList = append(akunList, admin)

	loadData()

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/register", registerHandler)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/admin", adminHandler)
	http.HandleFunc("/ownerr", ownerHandler)
	http.HandleFunc("/buyer", buyerHandler)
	http.HandleFunc("/transactions", transactionsHandler)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	fs := http.FileServer(http.Dir("img"))
	http.Handle("/img/", http.StripPrefix("/img/", fs))

	log.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "index", nil)
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		password := r.FormValue("password")
		tipe := r.FormValue("tipe")

		if tipe != "pemilik" && tipe != "pembeli" {
			http.Error(w, "Tipe akun tidak valid. Hanya pemilik dan pembeli yang diizinkan.", http.StatusBadRequest)
			return
		}

		akun := Akun{Username: username, Password: password, Tipe: tipe, Disetujui: false}
		akunList = append(akunList, akun)
		saveData()
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	renderTemplate(w, "register", nil)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		password := r.FormValue("password")

		for _, akun := range akunList {
			if akun.Username == username && akun.Password == password {
				if !akun.Disetujui {
					http.Error(w, "Akun belum disetujui oleh admin.", http.StatusForbidden)
					return
				}
				switch akun.Tipe {
				case "admin":
					http.Redirect(w, r, "/admin", http.StatusSeeOther)
				case "pemilik":
					http.Redirect(w, r, "/ownerr", http.StatusSeeOther)
				case "pembeli":
					http.Redirect(w, r, "/buyer?username="+username, http.StatusSeeOther)
				default:
					http.Error(w, "Tipe akun tidak valid.", http.StatusForbidden)
				}
				return
			}
		}
		http.Error(w, "Login gagal. Username atau password salah.", http.StatusUnauthorized)
		return
	}
	renderTemplate(w, "login", nil)
}

func adminHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		action := r.FormValue("action")
		switch action {
		case "approve":
			index, err := strconv.Atoi(r.FormValue("index"))
			if err != nil {
				http.Error(w, "Invalid index", http.StatusBadRequest)
				return
			}
			if index < 0 || index >= len(akunList) {
				http.Error(w, "Index out of range", http.StatusBadRequest)
				return
			}
			akunList[index].Disetujui = true
			saveData()
		case "delete":
			index, err := strconv.Atoi(r.FormValue("index"))
			if err != nil {
				http.Error(w, "Invalid index", http.StatusBadRequest)
				return
			}
			if index < 0 || index >= len(akunList) {
				http.Error(w, "Index out of range", http.StatusBadRequest)
				return
			}
			akunList = append(akunList[:index], akunList[index+1:]...)
			saveData()
		}
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}
	renderTemplate(w, "admin", akunList)
}

func ownerHandler(w http.ResponseWriter, r *http.Request) {
	action := r.URL.Query().Get("action")

	switch action {
	case "add":
		if r.Method == http.MethodPost {
			nama := r.FormValue("nama")
			harga, err := strconv.Atoi(r.FormValue("harga"))
			if err != nil {
				log.Println("Error converting harga:", err)
			}
			kuantitas, err := strconv.Atoi(r.FormValue("kuantitas"))
			if err != nil {
				log.Println("Error converting kuantitas:", err)
			}

			log.Printf("Received data - Nama: %s, Harga: %d, Kuantitas: %d", nama, harga, kuantitas)

			barang := Barang{Nama: nama, Harga: harga, Kuantitas: kuantitas}
			barangList = append(barangList, barang)
			saveData()
			http.Redirect(w, r, "/ownerr", http.StatusSeeOther)
			return
		}

	case "edit":
		if r.Method == http.MethodPost {
			editNama := r.FormValue("edit_nama")
			newNama := r.FormValue("new_nama")
			newHarga, err := strconv.Atoi(r.FormValue("new_harga"))
			if err != nil {
				log.Println("Error converting new_harga:", err)
			}
			newKuantitas, err := strconv.Atoi(r.FormValue("new_kuantitas"))
			if err != nil {
				log.Println("Error converting new_kuantitas:", err)
			}

			log.Printf("Editing data - Nama Lama: %s, Nama Baru: %s, New Harga: %d, New Kuantitas: %d", editNama, newNama, newHarga, newKuantitas)

			for i, barang := range barangList {
				if barang.Nama == editNama {
					barangList[i].Nama = newNama
					barangList[i].Harga = newHarga
					barangList[i].Kuantitas = newKuantitas
					saveData()
					http.Redirect(w, r, "/ownerr", http.StatusSeeOther)
					return
				}
			}

			http.Error(w, "Barang not found", http.StatusBadRequest)
			return
		}

	case "delete":
		if r.Method == http.MethodPost {
			deleteNama := r.FormValue("delete_nama")
			log.Printf("Deleting data - Nama: %s", deleteNama)

			for i, barang := range barangList {
				if barang.Nama == deleteNama {
					barangList = append(barangList[:i], barangList[i+1:]...)
					saveData()
					http.Redirect(w, r, "/ownerr", http.StatusSeeOther)
					return
				}
			}

			http.Error(w, "Barang not found", http.StatusBadRequest)
			return
		}
	}

	data := OwnerData{
		BarangList:    barangList,
		TransaksiList: transaksiList,
	}
	renderTemplate(w, "ownerr", data)
}


func buyerHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method == http.MethodPost {
        namaBarang := r.FormValue("nama_barang")
        jumlah, err := strconv.Atoi(r.FormValue("jumlah"))
        if err != nil {
            http.Error(w, "Jumlah tidak valid", http.StatusBadRequest)
            return
        }
        username := r.FormValue("username")

        // Cari barang berdasarkan nama
        var barangDitemukan *Barang
        for i := range barangList {
            if barangList[i].Nama == namaBarang {
                barangDitemukan = &barangList[i]
                break
            }
        }

        if barangDitemukan == nil {
            http.Error(w, "Barang tidak ditemukan", http.StatusBadRequest)
            return
        }

        if jumlah > barangDitemukan.Kuantitas {
            http.Error(w, "Kuantitas barang tidak mencukupi", http.StatusBadRequest)
            return
        }

        // Update kuantitas barang
        barangDitemukan.Kuantitas -= jumlah
        if barangDitemukan.Kuantitas == 0 {
            // Jika kuantitas barang habis, hapus dari list
            for i, b := range barangList {
                if b.Nama == barangDitemukan.Nama {
                    barangList = append(barangList[:i], barangList[i+1:]...)
                    break
                }
            }
        }

        // Tambahkan transaksi
        transaksi := Transaksi{UsernamePembeli: username, BarangDibeli: *barangDitemukan, Jumlah: jumlah}
        transaksiList = append(transaksiList, transaksi)
        saveData()
        http.Redirect(w, r, "/buyer?username="+username, http.StatusSeeOther)
        return
    }

    // Ambil username dari query parameter
    username := r.URL.Query().Get("username")
    search := r.URL.Query().Get("search")
    sortBy := r.URL.Query().Get("sort")

    filteredBarangList := barangList

    // Filter barang berdasarkan pencarian
    if search != "" {
        var tempBarangList []Barang
        for _, barang := range filteredBarangList {
            if strings.Contains(strings.ToLower(barang.Nama), strings.ToLower(search)) {
                tempBarangList = append(tempBarangList, barang)
            }
        }
        filteredBarangList = tempBarangList
    }

    // Urutkan barang berdasarkan pilihan sort
    switch sortBy {
    case "name_asc":
        sort.Slice(filteredBarangList, func(i, j int) bool {
            return filteredBarangList[i].Nama < filteredBarangList[j].Nama
        })
    case "name_desc":
        sort.Slice(filteredBarangList, func(i, j int) bool {
            return filteredBarangList[i].Nama > filteredBarangList[j].Nama
        })
    case "price":
        sort.Slice(filteredBarangList, func(i, j int) bool {
            return filteredBarangList[i].Harga < filteredBarangList[j].Harga
        })
    case "price_desc":
        sort.Slice(filteredBarangList, func(i, j int) bool {
            return filteredBarangList[i].Harga > filteredBarangList[j].Harga
        })
    }

    data := BuyerData{
        Username:   username,
        BarangList: filteredBarangList,
    }
    renderTemplate(w, "buyer", data)
}


func transactionsHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "transactions", transaksiList)
}

func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	t, ok := templatesCache[tmpl]
	if !ok {
		http.Error(w, "Template not found.", http.StatusInternalServerError)
		return
	}
	err := t.Execute(w, data)
	if err != nil {
		http.Error(w, "Error rendering template.", http.StatusInternalServerError)
	}
}
