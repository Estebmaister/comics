package repository

import (
	"comics/domain"
	"context"
	"database/sql"
	"reflect"
	"testing"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

func TestNewSQLiteDB(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
		want    *SQLiteDB
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewSQLiteDB(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewSQLiteDB() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewSQLiteDB() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSQLiteDB_InitDatabase(t *testing.T) {
	type fields struct {
		DB    *sql.DB
		Drive string
		Path  string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &SQLiteDB{
				DB:    tt.fields.DB,
				Drive: tt.fields.Drive,
				Path:  tt.fields.Path,
			}
			if err := db.InitDatabase(); (err != nil) != tt.wantErr {
				t.Errorf("SQLiteDB.InitDatabase() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSQLiteDB_Close(t *testing.T) {
	type fields struct {
		DB    *sql.DB
		Drive string
		Path  string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &SQLiteDB{
				DB:    tt.fields.DB,
				Drive: tt.fields.Drive,
				Path:  tt.fields.Path,
			}
			if err := db.Close(); (err != nil) != tt.wantErr {
				t.Errorf("SQLiteDB.Close() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSQLiteDB_FindAll(t *testing.T) {
	type fields struct {
		DB    *sql.DB
		Drive string
		Path  string
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []domain.User
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &SQLiteDB{
				DB:    tt.fields.DB,
				Drive: tt.fields.Drive,
				Path:  tt.fields.Path,
			}
			got, err := db.FindAll(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("SQLiteDB.FindAll() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SQLiteDB.FindAll() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSQLiteDB_FindById(t *testing.T) {
	type fields struct {
		DB    *sql.DB
		Drive string
		Path  string
	}
	type args struct {
		ctx context.Context
		id  uuid.UUID
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *domain.User
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &SQLiteDB{
				DB:    tt.fields.DB,
				Drive: tt.fields.Drive,
				Path:  tt.fields.Path,
			}
			got, err := db.FindById(tt.args.ctx, tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("SQLiteDB.FindById() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SQLiteDB.FindById() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSQLiteDB_Create(t *testing.T) {
	type fields struct {
		DB    *sql.DB
		Drive string
		Path  string
	}
	type args struct {
		ctx  context.Context
		user *domain.User
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &SQLiteDB{
				DB:    tt.fields.DB,
				Drive: tt.fields.Drive,
				Path:  tt.fields.Path,
			}
			if err := db.Create(tt.args.ctx, tt.args.user); (err != nil) != tt.wantErr {
				t.Errorf("SQLiteDB.Create() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSQLiteDB_Update(t *testing.T) {
	type fields struct {
		DB    *sql.DB
		Drive string
		Path  string
	}
	type args struct {
		ctx  context.Context
		user *domain.User
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &SQLiteDB{
				DB:    tt.fields.DB,
				Drive: tt.fields.Drive,
				Path:  tt.fields.Path,
			}
			if err := db.Update(tt.args.ctx, tt.args.user); (err != nil) != tt.wantErr {
				t.Errorf("SQLiteDB.Update() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSQLiteDB_Delete(t *testing.T) {
	type fields struct {
		DB    *sql.DB
		Drive string
		Path  string
	}
	type args struct {
		ctx context.Context
		id  uuid.UUID
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &SQLiteDB{
				DB:    tt.fields.DB,
				Drive: tt.fields.Drive,
				Path:  tt.fields.Path,
			}
			if err := db.Delete(tt.args.ctx, tt.args.id); (err != nil) != tt.wantErr {
				t.Errorf("SQLiteDB.Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
