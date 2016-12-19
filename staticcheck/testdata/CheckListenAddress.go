package pkg

import "net/http"

func fn() {
	// Seen in actual code
	http.ListenAndServe("localhost:8080/", nil) // MATCH /invalid port or service name in host:port pair/
	http.ListenAndServe("localhost", nil)       // MATCH /missing port in address localhost/
	http.ListenAndServe("localhost:8080", nil)
	http.ListenAndServe(":8080", nil)
	http.ListenAndServe(":http", nil)
	http.ListenAndServe("localhost:http", nil)
	http.ListenAndServe("local_host:8080", nil)
}
