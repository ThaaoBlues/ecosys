# How do I let Qsync know that I want to work with it ?
You have to send an intent to qsync. Qsync will then handle the creation of the sync task and app's folder.
Don't try to fool Us by placing another thing than your app's package name
or when you will be using the content provider it will not do anything.
```java
Intent sendIntent = new Intent();
sendIntent.setAction(Intent.ACTION_SYNC);
sendIntent.putExtra("action_flag", "[INSTALL_APP]");
sendIntent.putExtra("package_name", "your_app_package_name");
sendIntent.setType("text/plain");

if (sendIntent.resolveActivity(getPackageManager()) != null) {
    startActivity(sendIntent);
}
```




# How do I let Qsync sync the files of my android app ?
Qsync uses ContentProvider to let your app write the files and folders you want to synchronize.
examples :
> Using these functions, you must use the appropriate URI, like 'content://com.qsync.fileprovider/my_app_package_name/subfolder_if_i_want'

## I want to create a file in a folder named after my app :

```java
public void createFile(Uri rootDir, String fileName) {
    ContentValues values = new ContentValues();
    values.put(MediaStore.MediaColumns.DISPLAY_NAME, fileName);
    values.put(MediaStore.MediaColumns.MIME_TYPE, "text/plain");


    Uri newFileUri = getContentResolver().insert(rootDir, values);

    if (newFileUri != null) {
        Log.d(TAG, "File created successfully: " + newFileUri);
    } else {
        Log.e(TAG, "Failed to create file : " + fileName);
    }
    
}

createFile(Uri.parse('content://com.qsync.fileprovider/my_app_package_name'),'myfile.txt');
```


## I want to create a folder
```java
public void createFolder(Uri rootUri, String folderName) {
    ContentValues values = new ContentValues();
    values.put(MediaStore.MediaColumns.DISPLAY_NAME, folderName);
    values.put(MediaStore.MediaColumns.MIME_TYPE, "vnd.android.document/directory");

    Uri newFolderUri = getContentResolver().insert(rootUri, values);

    if (newFolderUri != null) {
        Log.d(TAG, "Folder created successfully: " + newFolderUri);
    } else {
        Log.e(TAG, "Failed to create folder: " + folderName);
    }
}

```


## I want to delete a file/folder
```java

public void deleteFile(Uri fileUri) {
    int rowsDeleted = getContentResolver().delete(fileUri, null, null);
    if (rowsDeleted > 0) {
        Log.d(TAG, "File deleted successfully");
    } else {
        Log.e(TAG, "Failed to delete file");
    }
}

```


## I want to write a file

```java
public void writeFile(Context context, Uri fileUri) {
    try (ParcelFileDescriptor pfd = context.getContentResolver().openFileDescriptor(fileUri, "w");
         FileOutputStream outputStream = new FileOutputStream(pfd.getFileDescriptor())) {

        // Write to the output stream
        String data = "Hello, World!";
        outputStream.write(data.getBytes());

    } catch (IOException e) {
        e.printStackTrace();
    }
}

```

## I want to read a file

```java
public void readFile(Context context, Uri fileUri) {
    try (ParcelFileDescriptor pfd = context.getContentResolver().openFileDescriptor(fileUri, "r");
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

> All these methods dont create the file if it does not exists. For that, user insert()



## The implementation of the Qsync file provider is visible on [the QSync mobile repo](https://github.com/ThaaoBlues/qsync_mobile/blob/master/app/src/main/java/com/qsync/qsync/FileProvider.java).