package server

var goticFiles map[string]string

func init() {
  goticFiles = make(map[string]string)

  goticFiles["db/queries.sql"] = "-- name: load-user-by-token\nSELECT * from users where token = $1\n"

}
