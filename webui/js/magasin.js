/*
 * @file            webui/js/magasin.js
 * @description     
 * @author          thaaoblues <thaaoblues81@gmail.com>
 * @createTime      2024-08-23 16:56:16
 * @lastModified    2024-08-23 17:30:47
 * Copyright ©Théo Mougnibas All rights reserved
*/

var os = navigator.platform.split(" ")[0]


function openSection(evt, sectionName) {
    var i, tabcontent, tablinks;
    tabcontent = document.getElementsByClassName("section");
    for (i = 0; i < tabcontent.length; i++) {
        tabcontent[i].style.display = "none";
    }
    tablinks = document.getElementsByClassName("tablinks");
    for (i = 0; i < tablinks.length; i++) {
        tablinks[i].className = tablinks[i].className.replace(" active", "");
    }
    document.getElementById(sectionName).style.display = "block";
    evt.currentTarget.className += " active";
}



// Fetch and parse the config file
fetch('https://raw.githubusercontent.com/ThaaoBlues/ecosys/master/magasin_database.json')
    .then(response => response.json())
    .then(data => {
        // Process data and generate HTML content dynamically
        generateHtml(data);
    })
    .catch(error => console.error('Error fetching config file:', error));

function generateHtml(data) {
// Extract tout_en_un_configs and grapin_configs from data
var toutEnUnConfigs = data.tout_en_un_configs;
var grapinConfigs = data.grapin_configs;
console.log(data);

// Generate HTML content for Tout en un section
var toutEnUnContainer = document.getElementById('ToutEnUn');
toutEnUnConfigs.forEach(config => {

    if(config.SupportedPlatforms.indexOf(os) > -1){
        var card = document.createElement('div');
        card.className = 'card';

        var image = document.createElement('img');
        image.src = config.AppIconURL;
        image.alt = 'App Image';

        var cardContent = document.createElement('div');
        cardContent.className = 'card-content';

        var cardTitle = document.createElement('div');
        cardTitle.className = 'card-title';
        cardTitle.textContent = config.AppName;

        var cardDescription = document.createElement('div');
        cardDescription.className = 'card-description';
        cardDescription.textContent = config.AppDescription;

        var installButton = document.createElement('button');
        installButton.textContent = 'Install app';
        installButton.className = 'install-button'; // You can add a class to style the button
        installButton.onclick = function() {
            showLoadingPopup();                    
            fetch('/install-tout-en-un', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(config),
            })
            .then(response => response.json())
            .then(data => hideLoadingPopup())
            .catch((error) => console.error('Error:', error));
        };

        cardContent.appendChild(cardTitle);
        cardContent.appendChild(cardDescription);
        cardContent.appendChild(installButton); // Add the button to the card content

        card.appendChild(image);
        card.appendChild(cardContent);

        toutEnUnContainer.appendChild(card);
    }
    
});

// Generate HTML content for Grapins section
var grapinsContainer = document.getElementById('Grapins');
grapinConfigs.forEach(config => {

    if(config.SupportedPlatforms.indexOf(os) > -1){

        var card = document.createElement('div');
        card.className = 'card';

        var image = document.createElement('img');
        image.src = 'https://via.placeholder.com/300';
        image.alt = 'App Image';

        var cardContent = document.createElement('div');
        cardContent.className = 'card-content';

        var cardTitle = document.createElement('div');
        cardTitle.className = 'card-title';
        cardTitle.textContent = config.AppName;

        var cardDescription = document.createElement('div');
        cardDescription.className = 'card-description';
        cardDescription.textContent = `Description of ${config.AppName}.`;

        var installButton = document.createElement('button');
        installButton.textContent = 'Install Grapin';
        installButton.className = 'install-button'; // You can add a class to style the button

        installButton.onclick = function() {
            showLoadingPopup();
            fetch('/install-grapin', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(config),
            })
            .then(response => response.json())
            .then(data => console.log(data))
            .catch((error) => console.error('Error:', error));
            
        };


        cardContent.appendChild(cardTitle);
        cardContent.appendChild(cardDescription);
        cardContent.appendChild(installButton); // Add the button to the card content

        card.appendChild(image);
        card.appendChild(cardContent);

        grapinsContainer.appendChild(card);

    }
});
}



// Function to show the loading animation popup
function showLoadingPopup() {
document.getElementById('loading-animation-popup').style.display = 'block';
}

// Function to hide the loading animation popup
function hideLoadingPopup() {
document.getElementById('loading-animation-popup').style.display = 'none';
}