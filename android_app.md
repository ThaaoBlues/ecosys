# How do I let Ecosys know that I want to work with it ?
You have to send an intent to Ecosys. Ecosys will then handle the creation of the sync task and app's folder.
```java
 Intent intent = new Intent(Intent.ACTION_SYNC);
        intent.setClassName("com.ecosys.ecosys","com.ecosys.ecosys.AppsIntentActivity");
        intent.putExtra("action_flag","[INSTALL_APP]");
        intent.putExtra("package_name",getContext().getPackageName());

startActivity(intent);
```



> You need to implement an activity NAMED EcosysCallbackActivity that recieve Intent.ACTION_SEND so Ecosys can send you the Uri back in the intent data with the proper permissions

## Don't forget to retrieve the intent permissions

```java
Intent intent = getIntent();
Uri uri = intent.getData();

if (uri != null && intent.getExtra("flag").equals("[INSTALL_APP]")) {
    // Take persistable URI permission
    final int takeFlags = intent.getFlags()
            & (Intent.FLAG_GRANT_READ_URI_PERMISSION | Intent.FLAG_GRANT_WRITE_URI_PERMISSION);
    getContentResolver().takePersistableUriPermission(uri, takeFlags);

    // Now you can work with the URI
    // For example, you can list files, open input/output streams, etc.
}

```


# How do I let Ecosys sync the files of my android app ?
Ecosys uses ContentProvider to let your app write the files and folders you want to synchronize.
examples :
> Using these functions, you must use the appropriate URI, like 'content://com.ecosys.ecosys.fileprovider/apps/my_app_package_name/subfolder_if_i_want'

## /!\ The content provider is not designed to share an entire directory, to bypass that limitation the file creation is a little more complicated : It will also be using EcosysCallbackActivity that recieve Intent.ACTION_SEND but with a [CREATE_FILE] flag.

> File creation : as app installation, uri is sent to EcosysCallbackActivity but very easily predictible so you can start using the file right after the Ecosys activity finished without actually storing the retrieved uri. 

> If you provide a relative path before the actual file name, necessary directories will be created if they do not exists
```java
public void checkFileCreated(String fileName) {

        Intent intent = new Intent(Intent.ACTION_SYNC);
        intent.setClassName("com.ecosys.ecosys","com.ecosys.ecosys.AppsIntentActivity");
        intent.putExtra("action_flag","[CREATE_FILE]");
        intent.putExtra("package_name",getContext().getPackageName());
        intent.putExtra("file_path","subdirectory_that_will_be_created/"+fileName);
        intent.putExtra("mime_type","text/plain");
        Log.d(TAG,"starting activity with sync intent");
        startActivity(intent);
}

```


> I implemented the same thing for full directories path, same usage as before, just with [CREATE_DIRECTORY] flag
```java
public void checkFileCreated(String fileName) {

        Intent intent = new Intent(Intent.ACTION_SYNC);
        intent.setClassName("com.ecosys.ecosys","com.ecosys.ecosys.AppsIntentActivity");
        intent.putExtra("action_flag","[CREATE_DIRECTORY]");
        intent.putExtra("package_name",getContext().getPackageName());
        intent.putExtra("file_path","subdirectory_that_will_be_created/sub_sub_another_one");
        Log.d(TAG,"starting activity with sync intent");
        startActivity(intent);
}

```


> Callback called by Ecosys via intent after it created your file
```java

Intent intent = getIntent();
Uri uri = intent.getData();

if (uri != null && intent.getExtra("action_flag").equals("[CREATE_FILE]")) {
    // Take persistable URI permission
    final int takeFlags = intent.getFlags()
            & (Intent.FLAG_GRANT_READ_URI_PERMISSION | Intent.FLAG_GRANT_WRITE_URI_PERMISSION);
    getContentResolver().takePersistableUriPermission(uri, takeFlags);

    // Now you can work with the URI
    // For example, you can list files, open input/output streams, etc.
}

```

## All the rest of the files manipulations are by using the regular content provider at content://com.Ecosys.Ecosys.fileprovider/apps/my_app_package_name

## Examples :

### I want to write a file

```java
public void writeFile(Context context, Uri fileUri) {
    try (ParcelFileDescriptor pfd = context.getContentResolver().openFile(fileUri, "w");
         FileOutputStream outputStream = new FileOutputStream(pfd.getFileDescriptor())) {

        // Write to the output stream
        String data = "Hello, World!";
        outputStream.write(data.getBytes());

    } catch (IOException e) {
        e.printStackTrace();
    }
}

```

### I want to read a file

```java
public void readFile(Context context, Uri fileUri) {
    try (ParcelFileDescriptor pfd = context.getContentResolver().openFile(fileUri, "r");
         FileInputStream inputStream = new FileInputStream(pfd.getFileDescriptor())) {

        // Read from the input stream
        byte[] buffer = new byte[1024];
        int length;
        while ((length = inputStream.read(buffer)) > 0) {
            // Process the buffer
        }

    } catch (IOException e) {
        e.printStackTrace();
    }
}

```

# openInputStream() and openOutputStream() are also available !
