var socket = new WebSocket('ws://localhost:8275/ws');

// Handle tab visibility change
document.addEventListener('visibilitychange', function() {
    if (document.visibilityState === 'hidden') {
        // Tab is hidden, close the WebSocket connection
        socket.close();
        socket = null;
        console.log('WebSocket connection closed due to tab switch.');
    } else if (document.visibilityState === 'visible') {
        // Tab is visible again, reopen the WebSocket connection
        socket = new WebSocket('ws://localhost:8275/ws');
        console.log('Reopening websocket after user went back to tab.');
    }
});


socket.onopen = function(event) {
   //alert("connected");
};


async function openTaskChooserToLinkApp(){
    let title = document.createElement("h1");
    title.innerText = translations[currentLang].taskActions;
    popup.appendChild(title);

    const response = await sendRequest('/list-tasks');
    let btn = document.createElement("button");
    
    response.forEach((task, index) => {

        if(task.IsApp){
            const button = document.createElement('button');
            button.innerText = task.Name;
            task.Flag = "[APP_TO_LINK_CHOOSEN]";
            button.onclick = () => sendMessage(JSON.stringify(task));
            devicesDiv.appendChild(button);
        }

    });
    

    popup.appendChild(btn);
    showPopup();
}

socket.onmessage = function(event) {
    const parts = event.data.split("|")
    console.log("websocket event : "+parts);
    const flag = parts[0];
    const msg = parts[1];
    switch(flag){
        case "[OTDL]":
            let rep = confirm(msg);
            sendMessage(rep);     
            break;

        case "[MOTDL]":
            rep = confirm(msg);
            sendMessage(rep);    
            break;
        case "[CHOOSELINKPATH]":
            rep = confirm(msg);
            // don't send anything here, the backend will open a folder picker
            break;

        case "[ALERT_USER]":
            alert(msg)
            break;

        case "[CHOOSE_APP_TO_LINK]":
            openTaskChooserToLinkApp();
            break;

        case "[STOP_LOADING_ANIMATION]":
            document.getElementById('loading-animation-popup').style.display = 'none';
            break;
        case "success":
            break;
    }

};

socket.onerror = function(event) {
};

socket.onclose = function(event) {
};



function sendMessage(message) {
    socket.send(message);
}