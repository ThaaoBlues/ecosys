# How do I let Qsync know that I want to work with it ?
You have to send an intent to qsync, containing the application JSON string :




# How do I let Qsync sync the files of my android app ?
Qsync uses ContentProvider to let your app write the files and folders you want to synchronize.
examples :
> Using these functions, you must use the appropriate URI, like 'content://com.qsync.fileprovider/my_app_name/subfolder_if_i_want'

## I want to create a file in a folder named after my app :

```java
public void createFileInFolder(Uri folderUri, String fileName, String fileContent) {
    ContentValues values = new ContentValues();
    values.put(MediaStore.MediaColumns.DISPLAY_NAME, fileName);
    values.put(MediaStore.MediaColumns.MIME_TYPE, "text/plain");

    Uri newFileUri = Uri.withAppendedPath(folderUri, fileName);

    try {
        OutputStream outputStream = getContentResolver().openOutputStream(newFileUri);
        if (outputStream != null) {
            outputStream.write(fileContent.getBytes());
            outputStream.close();
            Log.d(TAG, "File created successfully: " + newFileUri);
        } else {
            Log.e(TAG, "Failed to open output stream for: " + newFileUri);
        }
    } catch (IOException e) {
        e.printStackTrace();
        Log.e(TAG, "IOException: " + e.getMessage());
    }
}

createFileInFolder('content://com.qsync.fileprovider/my_app_name','myfile.txt','this is the text content of the file');
```


## I want to write a non-text file :
```java

public void createImageFileInFolder(Uri folderUri, String fileName, Bitmap bitmap) {
    ContentValues values = new ContentValues();
    values.put(MediaStore.MediaColumns.DISPLAY_NAME, fileName);
    values.put(MediaStore.MediaColumns.MIME_TYPE, "image/jpeg");

    Uri newFileUri = Uri.withAppendedPath(folderUri, fileName);

    try {
        OutputStream outputStream = getContentResolver().openOutputStream(newFileUri);
        if (outputStream != null) {
            bitmap.compress(Bitmap.CompressFormat.JPEG, 100, outputStream);
            outputStream.close();
            Log.d(TAG, "Image file created successfully: " + newFileUri);
        } else {
            Log.e(TAG, "Failed to open output stream for: " + newFileUri);
        }
    } catch (IOException e) {
        e.printStackTrace();
        Log.e(TAG, "IOException: " + e.getMessage());
    }
}
```

## I want to create a folder
```java
public void createFolder(Uri folderUri, String folderName) {
    ContentValues values = new ContentValues();
    values.put(MediaStore.MediaColumns.DISPLAY_NAME, folderName);
    values.put(MediaStore.MediaColumns.MIME_TYPE, "vnd.android.document/directory");

    Uri newFolderUri = getContentResolver().insert(folderUri, values);

    if (newFolderUri != null) {
        Log.d(TAG, "Folder created successfully: " + newFolderUri);
    } else {
        Log.e(TAG, "Failed to create folder: " + folderName);
    }
}

```

## I want to delete a file
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

## I want to delete a folder

```java
public void deleteFolder(Uri folderUri) {
    int rowsDeleted = getContentResolver().delete(folderUri, null, null);
    if (rowsDeleted > 0) {
        Log.d(TAG, "Folder deleted successfully");
    } else {
        Log.e(TAG, "Failed to delete folder");
    }
}

```

## I want to modify a text file

```java
public void modifyTextFile(Uri fileUri, String newText) {
    try {
        OutputStream outputStream = getContentResolver().openOutputStream(fileUri);
        if (outputStream != null) {
            outputStream.write(newText.getBytes());
            outputStream.close();
            Log.d(TAG, "Text file modified successfully");
        } else {
            Log.e(TAG, "Failed to open output stream for: " + fileUri);
        }
    } catch (IOException e) {
        e.printStackTrace();
        Log.e(TAG, "IOException: " + e.getMessage());
    }
}

```

## The implementation of the Qsync file provider is visible at [].