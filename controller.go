// Main logic/functionality for the web application.
// This is where you need to implement your own server.
package main

// Reminder that you're not allowed to import anything that isn't part of the Go standard library.
// This includes golang.org/x/
import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
	"regexp"
	_ "strconv"

	log "github.com/sirupsen/logrus"
)

func processRegistration(response http.ResponseWriter, request *http.Request) {
	username := request.FormValue("username")
	password := request.FormValue("password")

	// Check if username already exists
	row := db.QueryRow("SELECT username FROM users WHERE username = ?", username)
	var savedUsername string
	err := row.Scan(&savedUsername)
	if err != sql.ErrNoRows {
		response.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(response, "username %s already exists", savedUsername)
		return
	}

	// Generate salt
	const saltSizeBytes = 16
	salt, err := randomByteString(saltSizeBytes)
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(response, err.Error())
		return
	}

	hashedPassword := hashPassword(password, salt)

	_, err = db.Exec("INSERT INTO users VALUES (NULL, ?, ?, ?)", username, hashedPassword, salt)

	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(response, err.Error())
		return
	}

	//create a folder
	err = os.Mkdir("files/" + username, 0744)
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(response, err.Error())
		return
	}

	// Set a new session cookie
	initSession(response, username)

	// Redirect to next page
	http.Redirect(response, request, "/", http.StatusFound)
}

func processLoginAttempt(response http.ResponseWriter, request *http.Request) {
	// Retrieve submitted values
	username := request.FormValue("username")
	password := request.FormValue("password")

	row := db.QueryRow("SELECT password, salt FROM users WHERE username = ?", username)

	// Parse database response: check for no response or get values
	var encodedHash, encodedSalt string
	err := row.Scan(&encodedHash, &encodedSalt)
	if err == sql.ErrNoRows {
		response.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(response, "unknown user")
		return
	} else if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(response, err.Error())
		return
	}

	// Hash submitted password with salt to allow for comparison
	submittedPassword := hashPassword(password, encodedSalt)

	// Verify password
	if submittedPassword != encodedHash {
		fmt.Fprintf(response, "incorrect password")
		return
	}

	// Set a new session cookie
	initSession(response, username)

	// Redirect to next page
	http.Redirect(response, request, "/", http.StatusFound)
}

func processLogout(response http.ResponseWriter, request *http.Request) {
	// get the session token cookie
	cookie, err := request.Cookie("session_token")
	// empty assignment to suppress unused variable warning
	_, _ = cookie, err

	// get username of currently logged in user
	username := getUsernameFromCtx(request)
	// empty assignment to suppress unused variable warning
	_ = username

	//////////////////////////////////
	// BEGIN TASK 2: YOUR CODE HERE
	//////////////////////////////////

	// TODO: clear the session token cookie in the user's browser
	// HINT: to clear a cookie, set its MaxAge to -1


	http.SetCookie(response, &http.Cookie{
		Name: "session_token",
		MaxAge: -1,
	})

	db.Exec("DELETE FROM sessions WHERE username = ?", username)

	// TODO: delete the session from the database

	//////////////////////////////////
	// END TASK 2: YOUR CODE HERE
	//////////////////////////////////

	// redirect to the homepage
	http.Redirect(response, request, "/", http.StatusSeeOther)
}

func processUpload(response http.ResponseWriter, request *http.Request, username string) {

	//////////////////////////////////
	// BEGIN TASK 3: YOUR CODE HERE
	//////////////////////////////////

	// HINT: files should be stored in const filePath = "./files"

	file, header, error := request.FormFile("file")
	if error != nil{
		//replace with actual error
		fmt.Fprintf(response, "Error uploading")
		return
	}
	_ = file
	filename := header.Filename

	re := regexp.MustCompile("^[a-zA-Z0-9\\.]{1,50}$")

	if !re.MatchString(filename){
		fmt.Fprintf(response, "Filename is Invalid")
		response.WriteHeader(http.StatusInternalServerError)
		return
	}

	//now add file to user
	db.Exec("INSERT INTO files VALUES (NULL, ?, ?, ?)", username, username, filename)

	//and get the new global id
	row := db.QueryRow("SELECT id FROM files WHERE username = ? AND filename = ?", username, filename)

	var fileID int

	err := row.Scan(&fileID)

	if err != nil {
		fmt.Fprintf(response, "Failure at retreive id")
		return
	}

	//fmt.Println("ID is ", fileID)


	//now load the file
	b, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	data := []byte(b)

	//create the file with name id

	fileloc := "./"+filepath.Join("files", username, filename)

	//fmt.Println("Filepath is ", fileloc)

	f, err := os.Create(fileloc)

	if err != nil{
		fmt.Println("Failure to make file")
		response.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(response, err.Error())
		return
	}

	defer f.Close()

	amount, err:= f.Write(data)

	if err != nil{
		//fmt.Println("Failure to write to file")
		response.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(response, err.Error())
		return
	}

	_ = amount
	//fmt.Printf("Wrote %d bytes\n", amount)


	// replace this statement
	fmt.Fprintf(response, "File uploaded!")

	//////////////////////////////////
	// END TASK 3: YOUR CODE HERE
	//////////////////////////////////
}

// fileInfo helps you pass information to the template
type fileInfo struct {
	Filename  string
	FileOwner string
	FilePath  string
}

func listFiles(response http.ResponseWriter, request *http.Request, username string) {
	files := make([]fileInfo, 0)

	//////////////////////////////////
	// BEGIN TASK 4: YOUR CODE HERE
	//////////////////////////////////

	// TODO: for each of the user's files, add a
	// corresponding fileInfo struct to the files slice.

	// replace this line
	//fmt.Fprintf(response, "placeholder")

	//////////////////////////////////
	// END TASK 4: YOUR CODE HERE
	//////////////////////////////////

	//
	rows, err := db.Query("SELECT id, filename, owner FROM files where username = ?", username)

	if err != nil{
		fmt.Fprintf(response, "Failure to load files due to Query, err:", err)
		response.WriteHeader(http.StatusBadRequest)
		return
	}

	defer rows.Close()

	for rows.Next(){
		var (
			id int
			filename string
			filestore fileInfo
			owner string
		)
		if err := rows.Scan(&id, &filename, &owner); err != nil {
			fmt.Fprintf(response, "Failure in scanning")
			response.WriteHeader(http.StatusBadRequest)
			return
		}
		//now make and add
		path := filepath.Join(".", "files", owner, filename)
		//fmt.Println("filename: " + filename + ", username: " + username + ", path: " + path)
		filestore.Filename = filename
		filestore.FileOwner = owner
		filestore.FilePath = path
		files = append(files, filestore)

	}

	data := map[string]interface{}{
		"Username": username,
		"Files":    files,
	}

	tmpl, err := template.ParseFiles("templates/base.html", "templates/list.html")
	if err != nil {
		log.Error(err)
	}
	err = tmpl.Execute(response, data)
	if err != nil {
		log.Error(err)
	}
}

func getFile(response http.ResponseWriter, request *http.Request, username string) {

	fileString := strings.TrimPrefix(request.URL.Path, "/file/")

	_ = fileString

	//fmt.Println("User " + username + "opening file at " + fileString)
	idString := strings.TrimPrefix(fileString, "files/")

	//now extract
	ownerId := strings.Split(idString, "/")
	owner := ownerId[0]

	filename := ownerId[1]

	//////////////////////////////////
	// BEGIN TASK 5: YOUR CODE HERE
	//////////////////////////////////

	// replace this line

	//fmt.Println("\n\nUser " + username + " is trying to access file at loc " + filename + " with path: " + fileString)

	//now query to see if the file is in their database - can only exist once
	// db.QueryRow is ok

	row := db.QueryRow("SELECT username FROM files WHERE username = ? AND filename = ? and owner = ?", username, filename, owner)

	_ = row

	var (
		user2 string
	)

	err := row.Scan(&user2)

	if err != nil{
		fmt.Fprintf(response, "User " + username + " does not have permission to view file " + filename + " owned by " + owner)
		response.WriteHeader(http.StatusInternalServerError)
		return
	}

	fmt.Println("File " + filename + " is owned by " + owner + ", downloading now...")


	//and serve
	//fmt.Fprintf(response, "File " + filename + " downloaded!")
	setNameOfServedFile(response, filename)
	http.ServeFile(response, request, fileString)



	//////////////////////////////////
	// END TASK 5: YOUR CODE HERE
	//////////////////////////////////
}

func setNameOfServedFile(response http.ResponseWriter, fileName string) {
	response.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
}

func processShare(response http.ResponseWriter, request *http.Request, sender string) {
	recipient := request.FormValue("username")
	filename := request.FormValue("filename")
	_ = filename

	if sender == recipient {
		response.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(response, "can't share with yourself")
		return
	}

	username := sender

	//////////////////////////////////
	// BEGIN TASK 6: YOUR CODE HERE
	//////////////////////////////////

	// first, make sure file exists

	row := db.QueryRow("SELECT id, owner FROM files WHERE username = ? AND filename = ?", username, filename)

	var(
		id int
		owner string
	)

	err := row.Scan(&id, &owner)

	if (err != nil){
		response.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(response, "You don't have access to a file with that name")
		return
	}


	db.Exec("INSERT INTO files VALUES (NULL, ?, ?, ?)", recipient, owner, filename)

	fmt.Fprint(response, "File shared!")

	//////////////////////////////////
	// END TASK 6: YOUR CODE HERE
	//////////////////////////////////

}

// Initiate a new session for the given username
func initSession(response http.ResponseWriter, username string) {
	// Generate session token
	sessionToken, err := randomByteString(16)
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(response, err.Error())
		return
	}

	expires := time.Now().Add(sessionDuration)

	// Store session in database
	_, err = db.Exec("INSERT INTO sessions VALUES (NULL, ?, ?, ?)", username, sessionToken, expires.Unix())
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(response, err.Error())
		return
	}

	// Set cookie with session data
	http.SetCookie(response, &http.Cookie{
		Name:     "session_token",
		Value:    sessionToken,
		Expires:  expires,
		SameSite: http.SameSiteStrictMode,
	})
}
