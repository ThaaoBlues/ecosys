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

function launchApp(appName) {
    // Function to launch the app
    alert("Launching " + appName + "...");
    fetch("/launch-app?AppId="+appId);
}

function deleteApp(appName,appId) {
    // Function to delete the app
    alert("Deleting " + appName + "...");
    fetch("/delete-app?AppId="+appId);
}