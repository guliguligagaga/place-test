let needsRedraw = false;
const originalCanvas = document.getElementById('originalCanvas');
const zoomedCanvas = document.getElementById('zoomedCanvas');
const backButton = document.getElementById('backButton');
const colorPicker = document.getElementById('colorPicker');
const originalCtx = originalCanvas.getContext('2d');
const zoomedCtx = zoomedCanvas.getContext('2d');
const width = 200;
const height = 200;
const applyButton = document.getElementById('applyButton');
const socket = new WebSocket('ws://localhost:8080/ws/');


var isZoomed = false;
const scale = 4;
var zoomScale = 20;
var zoomedX, zoomedY;

const colorMap = [
    [107, 1, 25],    // #6B0119
    [189, 0, 55],    // #BD0037
    [255, 69, 0],    // #FF4500
    [254, 168, 0],   // #FEA800
    [255, 212, 53],  // #FFD435
    [254, 248, 185], // #FEF8B9
    [1, 162, 103],   // #01A267
    [9, 204, 118],   // #09CC76
    [126, 236, 87],  // #7EEC57
    [2, 117, 93],    // #02756D
    [0, 157, 170],   // #009DAA
    [0, 204, 190],   // #00CCBE
    [36, 79, 164],   // #244FA4
    [55, 144, 234],  // #3790EA
    [82, 232, 243],  // #52E8F3
    [72, 57, 191],   // #4839BF
    [105, 91, 255],  // #695BFF
    [148, 179, 255], // #94B3FF
    [128, 29, 159],  // #801D9F
    [180, 73, 191],  // #B449BF
    [228, 171, 253], // #E4ABFD
    [221, 17, 126],  // #DD117E
    [254, 55, 129],  // #FE3781
    [254, 153, 169], // #FE99A9
    [109, 70, 47],   // #6D462F
    [155, 105, 38],  // #9B6926
    [254, 180, 112], // #FEB470
    [0, 0, 0],       // #000000
    [82, 82, 82],    // #525252
    [136, 141, 144], // #888D90
    [213, 214, 216], // #D5D6D8
    [255, 255, 255], // #FFFFFF
];
// Create an ArrayBuffer for the pixel data
let buffer = new ArrayBuffer(width * height * 4);
let pixelData = new Uint8ClampedArray(buffer);

function drawBoard() {
    if (needsRedraw) {
        // Create ImageData from the pixel data
        const imageData = new ImageData(pixelData, width, height);
        ctx.putImageData(imageData, 0, 0);
        needsRedraw = false;
    }
    requestAnimationFrame(drawBoard);
}

// Call this to start the drawing loop

async function drawCanvas() {
    const bitfield = await fetchCanvasData();
    const imageData = new ImageData(width, height);
    const data = imageData.data;

    const totalCells = width * height;

    for (let index = 0; index < totalCells; index++) {
        const byteIndex = Math.floor(index / 2); // Each byte stores 2 cells
        const bitOffset = (index % 2) * 4; // Each cell uses 4 bits

        // Extract the 4-bit color index
        const byte = bitfield[byteIndex];
        const colorIndex = (byte >> bitOffset) & 0xF;

        // Get the color from the palette
        const [r, g, b] = colorMap[colorIndex];

        // Calculate the position in the ImageData array
        const pos = index * 4; // Each pixel has 4 bytes (RGBA)
        data[pos] = r;     // Red
        data[pos + 1] = g; // Green
        data[pos + 2] = b; // Blue
        data[pos + 3] = 255; // Alpha (fully opaque)
    }
    originalCtx.putImageData(imageData, 0, 0);
}
async function fetchCanvasData() {
    const response = await fetch('http://localhost:8000/grid');
    const arrayBuffer = await response.arrayBuffer();
    return new Uint8Array(arrayBuffer);
}
socket.addEventListener('message', (event) => {
    const data = JSON.parse(event.data);
    const {x, y, color} = data;
    let pixelData = new Uint8ClampedArray(width * height * 4); // RGBA for each pixel
    if (x >= 0 && x < width && y >= 0 && y < height && color >= 0 && color < colorMap.length) {
        const index = (y * width + x) * 4;
        const colorArray = colorMap[color];
        pixelData[index] = colorArray[0];    // Red
        pixelData[index + 1] = colorArray[1]; // Green
        pixelData[index + 2] = colorArray[2]; // Blue
        pixelData[index + 3] = colorArray[3]; // Alpha

        // Update the canvas
        const imageData = new ImageData(pixelData, width, height);
        ctx.putImageData(imageData, 0, 0);
    }
});

// displayCanvas.addEventListener('click', function(event) {
//     // Get the position of the click
//     const rect = displayCanvas.getBoundingClientRect();
//     const selectedX = Math.ceil((event.clientX - rect.left) / scale);
//     const selectedY = Math.ceil((event.clientY - rect.top)/ scale);
//     console.log("x: " + selectedX + " y: " + selectedY)
//     // Get the color of the clicked pixel
//     const imageData = displayCtx.getImageData(selectedX, selectedY, 1, 1).data;
//     // Convert RGB to Hex
//     // Prefill the color picker with the clicked color
//     colorPicker.value = rgbToHex(imageData[0], imageData[1], imageData[2]);
//
//     // Show the color picker and apply button
//     colorPicker.style.display = 'block';
//     applyButton.style.display = 'block';
//     colorPicker.select();
// });

function rgbToHex(r, g, b) {
    return "#" + ((1 << 24) + (r << 16) + (g << 8) + b).toString(16).slice(1).toUpperCase();
}

// Handle Apply button click
applyButton.addEventListener('click', function() {
    const selectedColor = colorPicker.value;

    // Send the selected color to the backend
    fetch('/update-color/', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'X-CSRFToken': '{{ csrf_token }}'  // Ensure CSRF token is included
        },
        body: JSON.stringify({
            'x': selectedX,
            'y': selectedY,
            'color': selectedColor
        })
    })
        .then(response => response.json())
        .then(data => {
            console.log('Color update successful:', data);
            // Optionally, update the canvas with the new color
            ctx.fillStyle = selectedColor;
            ctx.fillRect(selectedX, selectedY, 1, 1);

            // Hide the color picker and apply button after use
            colorPicker.style.display = 'none';
            applyButton.style.display = 'none';
        })
        .catch((error) => {
            console.error('Error:', error);
        });
});

function zoomIntoArea(x, y) {
    zoomedCtx.imageSmoothingEnabled = false;
    var visibleWidth = zoomedCanvas.width / zoomScale;
    var visibleHeight = zoomedCanvas.height / zoomScale;

// Calculate the top-left corner of the cropped area to ensure the clicked pixel is centered
    var sx = x - visibleWidth / 2;
    var sy = y - visibleHeight / 2;

// Ensure cropping stays within bounds
    sx = Math.max(0, Math.min(sx, originalCanvas.width - visibleWidth));
    sy = Math.max(0, Math.min(sy, originalCanvas.height - visibleHeight));

// Draw the cropped and scaled image onto the zoomed canvas
    zoomedCtx.drawImage(originalCanvas, sx, sy, visibleWidth, visibleHeight, 0, 0, zoomedCanvas.width, zoomedCanvas.height);


    // Show the zoomed canvas and back button
    originalCanvas.style.display = 'none';
    zoomedCanvas.style.display = 'block';
    backButton.style.display = 'block';
}

function showColorPicker(x, y) {
    const imageData = zoomedCtx.getImageData(x, y, 1, 1).data;
    colorPicker.value = `#${((1 << 24) + (imageData[0] << 16) + (imageData[1] << 8) + imageData[2]).toString(16).slice(1).toUpperCase()}`;
    colorPicker.style.display = 'block';
    colorPicker.focus();
}

// Handle click on original canvas
originalCanvas.addEventListener('click', function(event) {
    if (isZoomed) return;

    const rect = originalCanvas.getBoundingClientRect();
    const x = Math.ceil((event.clientX - rect.left) / scale);
    const y = Math.ceil((event.clientY - rect.top)/ scale);
    console.log("x: " + x + " y: " + y)

    zoomedX = x - (originalCanvas.width / zoomScale) / 2;
    zoomedY = y - (originalCanvas.height / zoomScale) / 2;

    zoomIntoArea(zoomedX, zoomedY);
    isZoomed = true;
});

// Handle click on zoomed canvas
zoomedCanvas.addEventListener('click', function(event) {
    if (!isZoomed) return;

    var rect = zoomedCanvas.getBoundingClientRect();
    var x = event.clientX - rect.left;
    var y = event.clientY - rect.top;

    showColorPicker(x, y);
});

// Handle back button click
backButton.addEventListener('click', function() {
    zoomedCanvas.style.display = 'none';
    originalCanvas.style.display = 'block';
    backButton.style.display = 'none';
    colorPicker.style.display = 'none';
    isZoomed = false;
});
