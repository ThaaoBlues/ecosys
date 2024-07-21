const socket = new WebSocket('ws://localhost:8275/ws');

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