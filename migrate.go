/*
Package main - Database Migration Tool

This file is used solely for database migrations in development mode.
Developers can add or update models in "database/models.go" and then run
this migration tool to automatically create or update the corresponding tables
in the development database.

Usage:
 1. Make changes or add new models in "database/models.go".
 2. In this file, uncomment the main() function below.
 3. Run "go run migrate.go" to perform the migration.
 4. After the migration is complete, revert any changes to restore the standard
    application behavior (i.e., comment out main() here and uncomment main() in main.go).

Note:
- Ensure that the environment variables in your .env file are correctly set.
- This tool should only be used in a development environment.
*/
package main

import (
	"gradspaceBK/config"
	"gradspaceBK/database"
)

func Migrate() {
	config.LoadConfig() // Load .env first
	database.DBConnection()
	session := database.Session.Db
	database.MigrateDB(session)
}

// comment func main() in main.go and un comment below func main() to run migration
// go run migrate.go
// after running migration, comment below func main() and un comment func main() in main.go

// func main(){

// 	Migrate()
// }

