//app.js
const db = require('./database');
const express = require('express');
const cors = require('cors');
const app = express();

app.use(cors());
app.use(express.json());
const PORT = 3000;

// Endpoint to create a new watch party
app.post('/create-party', (req, res) => {
    const { name, adminName, password } = req.body;

    // Insert new party
    db.run(`INSERT INTO parties (name, admin_id, password) VALUES (?, ?, ?)`, [name, adminName, password], function(err) {
        if (err) {
            return res.status(500).json({ error: err.message });
        }

        // Insert the admin user with the new party's ID
        db.run(`INSERT INTO users (username, party_id, role) VALUES (?, ?, ?)`, [adminName, this.lastID, 'admin'], function(err) {
            if (err) {
                return res.status(500).json({ error: err.message });
            }
            res.json({ message: 'Party created successfully', partyId: this.lastID });
        });
    });
});

// Endpoint to join a watch party
app.post('/join-party', (req, res) => {
    const { partyId, username } = req.body;

    // Insert new user in the specified party
    db.run(`INSERT INTO users (username, party_id, role) VALUES (?, ?, ?)`, [username, partyId, 'user'], function(err) {
        if (err) {
            return res.status(500).json({ error: err.message });
        }
        res.json({ message: 'Joined party successfully', userId: this.lastID });
    });
});

// Endpoint to add a movie to a party
app.post('/add-movie', (req, res) => {
    const { title, releaseYear, posterPath, partyId } = req.body;

    db.run(`INSERT INTO movies (title, release_year, poster_path, party_id) VALUES (?, ?, ?, ?)`, [title, releaseYear, posterPath, partyId], function(err) {
        if (err) {
            return res.status(500).json({ error: err.message });
        }
        res.json({ message: 'Movie added successfully', movieId: this.lastID });
    });
});


app.listen(PORT, (error) =>{
        if(!error)
            console.log("Server is Successfully Running, and App is listening on port "+ PORT)
    else
        console.log("Error occurred, server can't start", error);
    }
);
