// database.js
const sqlite3 = require('sqlite3').verbose();

// Create a new SQLite database in memory or on disk
const db = new sqlite3.Database('./watch_party.db', (err) => {
    if (err) {
        console.error('Error opening database', err.message);
    } else {
        console.log('Connected to the SQLite database.');
    }
});

// Initialize tables if they don’t already exist
db.serialize(() => {
// Modify parties table to include a timestamp column
    db.run(`
        CREATE TABLE IF NOT EXISTS parties (
                                               id INTEGER PRIMARY KEY AUTOINCREMENT,
                                               name TEXT,
                                               admin_id TEXT,
                                               password TEXT,
                                               created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP  -- This adds a timestamp column
        )
    `);

    db.run(`
        CREATE TABLE IF NOT EXISTS users (
                                             id INTEGER PRIMARY KEY AUTOINCREMENT,
                                             username TEXT,
                                             party_id INTEGER,
                                             role TEXT,
                                             FOREIGN KEY (party_id) REFERENCES parties(id) ON DELETE CASCADE
            )
    `);

    db.run(`
        CREATE TABLE IF NOT EXISTS movies (
                                              id INTEGER PRIMARY KEY AUTOINCREMENT,
                                              title TEXT,
                                              release_year INTEGER,
                                              poster_path TEXT,
                                              party_id INTEGER,
                                              FOREIGN KEY (party_id) REFERENCES parties(id) ON DELETE CASCADE
            )
    `);

    db.run(`
        CREATE TABLE IF NOT EXISTS votes (
                                             id INTEGER PRIMARY KEY AUTOINCREMENT,
                                             user_id INTEGER,
                                             movie_id INTEGER,
                                             preference INTEGER,
                                             party_id INTEGER,
                                             FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
            FOREIGN KEY (movie_id) REFERENCES movies(id) ON DELETE CASCADE,
            FOREIGN KEY (party_id) REFERENCES parties(id) ON DELETE CASCADE
            )
    `);

});

module.exports = db;