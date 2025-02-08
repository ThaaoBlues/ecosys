var currentLang = 'fr';

function updateLanguage(lang) {
    currentLang = lang;
    document.getElementById('title').innerText = translations[lang].title;
    document.getElementById('no-internet-alert').innerText = translations[lang].noInternet;
    //document.getElementById('start-btn').innerText = translations[lang].startecosys;
    document.getElementById('create-sync-btn').innerText = translations[lang].createSyncTask;
    document.getElementById('open-magasin-btn').innerText = translations[lang].openMagasin;
    document.getElementById('toggle-largage-btn').innerText = translations[lang].toggleLargageAerien;
    document.getElementById('open-largages-btn').innerText = translations[lang].openLargagesFolder;
    document.getElementById('devices-title').innerText = translations[lang].devicesTitle;
    document.getElementById('tasks-title').innerText = translations[lang].tasksTitle;
}




async function sendRequest(url, method = 'GET', data = null) {
    let options = { method };
    if (data) {
        options.body = JSON.stringify(data);
        options.headers = { 'Content-Type': 'application/json' };
    }
    const response = await fetch(url, options);
    return response.json();
}

function updateResponse(message) {
    document.getElementById('response').innerText = message;
}

async function startecosys() {
    const response = await sendRequest('/start');
    updateResponse(response.message);
}

async function createlinkDevSyncTask() {

    var response = await sendRequest('/ask-file-path?is_folder='+true);

    if (response.Path) {
        response = await sendRequest(`/create?path=${encodeURIComponent(response.Path)}`);
        if(response.Message == "success"){
            alert(translations[currentLang].alertTaskCreated+response.Path);
        }
    }
}

async function linkDevice(task,device) {

    let data = {}
    data.SecureId = task.SecureId;
    data.IpAddr = device.ip_addr;
    data.DeviceId = device.device_id;
    console.log(data);

    const response = await sendRequest(`/link`,'POST',data);
    updateResponse(response.message);

}

async function listTasks() {
    const response = await sendRequest('/list-tasks');
    updateResponse(JSON.stringify(response, null, 2));
    const tasksDiv = document.getElementById('synchronisations');
    tasksDiv.innerHTML = "";

    let title = document.createElement("h2");
    title.innerText = translations[currentLang].tasksTitle;
    tasksDiv.appendChild(title)


    if(response != null){

        response.forEach((task, index) => {
            console.log(task);
            const button = document.createElement('button');
            if(task.IsApp){
                button.innerText = '( application ) '+task.Name;
            }else{
                button.innerText = task.Path;
            }
            
            button.onclick = () => openTasksActionsMenu(task);
            tasksDiv.appendChild(button);
        });

    }
}

async function listDevices() {
    const response = await sendRequest('/list-devices');
    const devicesDiv = document.getElementById('devices');

    devicesDiv.innerHTML = "";

    let title = document.createElement("h2");
    title.innerText = translations[currentLang].devicesTitle;
    devicesDiv.appendChild(title)


    if(response != null){
        response.forEach((device, index) => {
            const button = document.createElement('button');
            button.innerText = device.hostname;
            button.onclick = () => openNetworkDeviceActionsMenu(Object.assign({}, device));
            devicesDiv.appendChild(button);
        });
    }

}

async function openMagasin() {
    window.location.pathname = "/magasin";
}


async function toggleLargageAerien() {
    const response = await sendRequest('/toggle-largage');
}

async function sendLargage(device,folder=false) {
    let data = device;
    data.is_folder = folder;
    
    const response = await sendRequest('/ask-file-path?is_folder='+folder)
    data.filepath = response.Path;
   
    if(data.filepath != "[CANCELLED]"){
        const response = await sendRequest('/send-largage', 'POST', data);
    }

}



async function sendText(device) {

    let data = {};
    data.device = device;
    data.text = document.getElementById("custom-text").value;
    const response = await sendRequest('/send-text', 'POST', data);

}

function openSendTextOverlay(device){
    const popup = document.getElementById('popupContent');



    let title = document.createElement("h1");
    title.innerText = translations[currentLang].sendTextTitle;
    popup.appendChild(title);
    let wrapper = document.createElement("wrapper");
    wrapper.className = "grow-wrap";



    let textarea = document.createElement("textarea");
    textarea.setAttribute("id", "custom-text");
    textarea.style.maxHeight = "50vh"; // Set the maximum height
    textarea.style.maxWidth = "50vw";
    textarea.style.overflowY = "auto"; // Make the textarea scrollable when content exceeds the max height
        
    textarea.oninput = function () {
        textarea.parentNode.dataset.replicatedValue = textarea.value;
    };
    
    wrapper.appendChild(textarea);

    popup.appendChild(wrapper);

    let btn = document.createElement("button");
    btn.onclick = function (){sendText(device); closePopup();};
    btn.innerText = translations[currentLang].send;
    popup.appendChild(btn);

    showPopup()
}

function openNetworkDeviceActionsMenu(device){

    const popup = document.getElementById('popupContent');


    let title = document.createElement("h1");
    title.innerText = translations[currentLang].deviceActions;
    popup.appendChild(title);

    let btn = document.createElement("button");
    btn.onclick = function (){closePopup(); sendLargage(device,false)};
    btn.innerText = translations[currentLang].sendFile;
    popup.appendChild(btn);

    btn = document.createElement("button");
    btn.onclick = function(){closePopup(); sendLargage(device,true)};
    btn.innerText = translations[currentLang].sendFolder;
    popup.appendChild(btn);

    btn = document.createElement("button");
    btn.onclick = function(){closePopup();openSendTextOverlay(device);};
    btn.innerText = translations[currentLang].sendText;
    popup.appendChild(btn);
    showPopup();

}

function openTasksActionsMenu(task){
    const popup = document.getElementById('popupContent');


    let title = document.createElement("h1");
    title.innerText = translations[currentLang].taskActions;
    popup.appendChild(title);

    let btn = document.createElement("button");
    if(task.IsApp){
        btn.onclick = function (){closePopup();openApp(task);}
        btn.innerText = translations[currentLang].openApp;
        popup.appendChild(btn);

        btn = document.createElement("button");
        btn.onclick = function (){closePopup();chooseDeviceAndLinkIt(task);}
        btn.innerText = "Link as an App";
        popup.appendChild(btn);
    }

    btn = document.createElement("button");
    btn.onclick = function (){
        closePopup();
        task.IsApp = false;
        chooseDeviceAndLinkIt(task);
    }
    btn.innerText = translations[currentLang].syncAnotherDevice;
    popup.appendChild(btn);

    btn = document.createElement("button");
    btn.onclick = function(){closePopup();removeTask(task);};
    btn.innerText = translations[currentLang].removeTask;
    popup.appendChild(btn);

    btn = document.createElement("button");
    btn.onclick = function(){closePopup();toggleBackupMode(task);};
    if(task.BackupMode){
        btn.innerText = translations[currentLang].disableBackupMode;
    }else{
        btn.innerText = translations[currentLang].enableBackupMode;
    }
    popup.appendChild(btn);
    showPopup();
    
}



async function chooseDeviceAndLinkIt(task){

    const popup = document.getElementById('popupContent');


    let title = document.createElement("h1");
    title.innerText = "Choose a device to synchronize";
    popup.appendChild(title);

    const response = await sendRequest('/list-devices');

    if(response != null){
        response.forEach((device, index) => {
            const button = document.createElement('button');
            button.innerText = device.hostname;
            button.onclick = function (){
                linkDevice(task,device);
                closePopup();
            }
            popup.appendChild(button);
        });
    }

    showPopup();


    
}

async function removeTask(task){
    const response = await sendRequest('/remove-task?secure_id='+task.SecureId);

    if(response.message == "success"){
        alert("The sync task has been removed.");
    }else{
        alert("An error occured while removing the sync task.")
    }
}

async function toggleBackupMode(task){
    const response = await sendRequest('/toggle-backup-mode?secure_id='+task.SecureId);
    if(response.message == "success"){
        alert("Success !");
    }else{
        alert("An error occured while enabling/disabling backup mode for this sync task.")
    }
}


async function checkInternetConnection(){
    const response = await sendRequest('/check-internet');
    document.getElementById("no-internet-alert").hidden = response.ConnectionState;

}

function showPopup() {
    const overlay = document.getElementById('popupOverlay');
    overlay.classList.add('active');
}

function closePopup() {
    const overlay = document.getElementById('popupOverlay');
    overlay.classList.remove('active');
    // clean the popup content
    const popup = document.getElementById('popupContent');

    popup.innerHTML = "";

    let close = document.createElement("span");
    close.innerHTML = "&times;";
    close.onclick = closePopup;
    close.className = "close-btn";
    popup.appendChild(close);
}

window.onclick = function(event) {
    const overlay = document.getElementById('popupOverlay');
    if (event.target === overlay) {
        closePopup();
    }
}


async function openLargagesFolder(){
    const response = await sendRequest('/open-largages-folder');

}


async function openApp(task){
    const response = await sendRequest('/launch-app?AppId='+task.SecureId);

}

async function createSyncTask(){
    const response = await sendRequest('create-task');
}


addEventListener("DOMContentLoaded", (event) => {
    updateLanguage(navigator.language.split("-")[0]);

    // first call to not wait 5s at launch
    listDevices();
    listTasks();
    checkInternetConnection();


    setInterval(listDevices, 5000); // Update the device list every 5 seconds
    setInterval(listTasks, 5000); // Update the tasks list every 5 seconds
    setInterval(checkInternetConnection,5000);

    
});
