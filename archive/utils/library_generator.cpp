#include <string>
#include <iostream>
#include <cstdint>
#include <vector>
#include <sstream>
#include <fstream>

#include <sqlite3.h>

#include <boost/archive/text_oarchive.hpp>
#include <boost/archive/text_iarchive.hpp>
#include <boost/serialization/vector.hpp>

class Book {
    private:
        std::string title;
        std::string author;
        int year;
    public:
        Book() : title(""), author(""), year(0) {}
    
        Book(const std::string& title, const std::string& author, 
            int year)
            : title(title), author(author), year(year) {}
    
        template<class Archive>
        void serialize(Archive & ar, const unsigned int version) {
            ar & title;
            ar & author;
            ar & year;
        }

        std::string GetName() const { return title; }
        std::string GetTitle() const { return title; }
        std::string GetAuthor() const { return author; }
        int GetYear() const { return year; }
};

class Library {
    private:
        std::vector<Book*> books;
        bool isLoaded;
    public:
        void addBook(Book* book) {
            books.push_back(book);
        }
    
        void saveLibrary(const std::string& dbName) {
            sqlite3* db;

            std::string dbPath = "/tmp/" + dbName;

            if (sqlite3_open(dbPath.c_str(), &db) != SQLITE_OK) {
                std::cerr << "[-] Error opening database: " << sqlite3_errmsg(db) << std::endl;
                return;
            }
    
            const char* createTableSQL = "CREATE TABLE IF NOT EXISTS books (id INTEGER PRIMARY KEY, data TEXT);";

            if (sqlite3_exec(db, createTableSQL, nullptr, nullptr, nullptr) != SQLITE_OK) {
                std::cerr << "[-] Error creating table: " << sqlite3_errmsg(db) << std::endl;
                sqlite3_close(db);
                return;
            }

            for (const auto& book : books) {
                std::ostringstream archiveStream;
                boost::archive::text_oarchive archive(archiveStream);
                archive << book;
                std::string serializedData = archiveStream.str();
    
                std::string insertSQL = "INSERT INTO books (data) VALUES ('" + serializedData + "');";
                if (sqlite3_exec(db, insertSQL.c_str(), nullptr, nullptr, nullptr) != SQLITE_OK) {
                    std::cerr << "[-] Error inserting book: " << sqlite3_errmsg(db) << std::endl;
                }
            }
    
            sqlite3_close(db);
        }
};


Library* gLibrary = nullptr;

int main() {
    gLibrary = new Library();
    std::string option;
    std::string libraryName;
    std::cin >> libraryName;

    while (option != "exit") {
        std::string bookTitle, bookAuthor;
        int bookYear;
        std::cin >> bookTitle >> bookAuthor >> bookYear;
        
        gLibrary->addBook(new Book(bookTitle, bookAuthor, bookYear));
        std::cin >> option;
    }

    gLibrary->saveLibrary(libraryName);
    delete gLibrary;
}

