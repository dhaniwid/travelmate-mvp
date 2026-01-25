const http = require('http');

const data = JSON.stringify({
    origin: "Jakarta",
    destination: "Lombok",
    trip_days: 3,
    budget: 5000000,
    style: "Adventure"
});

const options = {
    hostname: 'localhost',
    port: 8889,
    path: '/api/v1/trips/stream',
    method: 'POST',
    headers: {
        'Content-Type': 'application/json',
        'Content-Length': data.length
    }
};

const req = http.request(options, (res) => {
    console.log('--- STREAM STARTED ---');
    res.on('data', (chunk) => {
        console.log('\x1b[36m%s\x1b[0m', '>> Received Chunk:'); // Warna Biru
        console.log(chunk.toString());
    });
    res.on('end', () => console.log('--- STREAM CLOSED ---'));
});

req.write(data);
req.end();