# GSA-Google-Assessment-Tool
This repository contains the source code for the Google Assessment Tool, a tool that performs audits over Google Workspace and Google Cloud Platform environments. It's part of a larger system for Google Workspace administration.

# Function: inventory
The `inventory` function serves to collect, organize, and store inventory data about Google Cloud projects, users, and groups within a Google Workspace environment.
1. The function initializes by creating a new `DirectoryAPI` instance. This instance facilitates interactions with the Google Admin SDK Directory API.
2. It retrieves all the groups in the Google Workspace domain by calling the `QueryGroups` method.
3. All the groups are then sorted by member count.
4. The function then loops over each group and, using goroutines for concurrency, fetches the group's owners, managers, members, and subscription emails. It also keeps track of the counts for each of these categories.
5. This information is then collected into a CSV format, with each row corresponding to a group and containing the group's email, member count, owner and manager counts along with their respective emails, and subscription count with corresponding emails.
6. Once all groups have been processed, the function creates a new CSV file named `groupsMap.csv` and writes all the collected information into it.
7. The function then loops over each user and, using goroutines for concurrency, fetches the user's primary email, account status, admin status, and other pertinent information.
8. This information is then collected into a CSV format, with each row corresponding to a user and containing the user's primary email, account status, admin status, and other pertinent information.
9. Once all users have been processed, the function creates a new CSV file named `users.csv` and writes all the collected information into it.
10. The function initializes by creating a new `CloudResourceManagerAPI` instance. This instance facilitates interactions with the Google Cloud Resource Manager API.
11. It retrieves all the Google Cloud projects in the Google Workspace domain by calling the `QueryProjects` method.
12. The function then loops over each project and, using goroutines for concurrency, fetches the project's name, ID, number, and service accounts.
13. This information is then collected into a CSV format, with each row corresponding to a project and containing the project's name, ID, number, and service accounts.
14. Once all projects have been processed, the function creates a new CSV file named `projects.csv` and writes all the collected information into it.
15. Finally, it calls the `uploadReport` function to upload the report to Google.

# Function: groupsAudit
The `groupsAudit` function performs an audit operation over all the groups in a Google domain. It's part of a larger system for Google Workspace administration. It retrieves and organizes detailed information about each group, including the groups' owners, managers, members, and subscription emails.
1. The function initializes by creating a new `DirectoryAPI` instance. This instance facilitates interactions with the Google Admin SDK Directory API.
2. It retrieves all the groups in the Google Workspace domain by calling the `QueryGroups` method.
3. All the groups are then sorted by member count.
4. The function then loops over each group and, using goroutines for concurrency, fetches the group's owners, managers, and subscription emails. It also keeps track of the counts for each of these categories.
5. This information is then collected into a CSV format, with each row corresponding to a group and containing the group's email, member count, owner and manager counts along with their respective emails, and subscription count with corresponding emails.
6. Once all groups have been processed, the function creates a new CSV file named `groupsMap.csv` and writes all the collected information into it.
7. Finally, it calls the `uploadReport` function to upload the report to Google.

# Function: usersAudit
The `usersAudit` function performs an audit operation over all the users in a Google domain. It's part of a larger system for Google Workspace administration. It retrieves and organizes detailed information about each user, including the user's primary email, account status, admin status, and other pertinent information.
1. The function initializes by creating a new `DirectoryAPI` instance. This instance facilitates interactions with the Google Admin SDK Directory API.
2. It retrieves all the users in the Google Workspace domain by calling the `QueryUsers` method.
3. The function then loops over each user and, using goroutines for concurrency, fetches the user's primary email, account status, admin status, and other pertinent information.
4. This information is then collected into a CSV format, with each row corresponding to a user and containing the user's primary email, account status, admin status, and other pertinent information.
5. Once all users have been processed, the function creates a new CSV file named `users.csv` and writes all the collected information into it.
6. Finally, it calls the `uploadReport` function to upload the report to Google.


# Function: projectsAudit
The `projectsAudit` function performs an audit operation over all the Google Cloud projects in a Google domain. It's part of a larger system for Google Workspace administration. It retrieves and organizes detailed information about each project, including the project's name, ID, number, and service accounts.
1. The function initializes by creating a new `CloudResourceManagerAPI` instance. This instance facilitates interactions with the Google Cloud Resource Manager API.
2. It retrieves all the Google Cloud projects in the Google Workspace domain by calling the `QueryProjects` method.
3. The function then loops over each project and, using goroutines for concurrency, fetches the project's name, ID, number, and service accounts.
4. This information is then collected into a CSV format, with each row corresponding to a project and containing the project's name, ID, number, and service accounts.
5. Once all projects have been processed, the function creates a new CSV file named `projects.csv` and writes all the collected information into it.
6. Finally, it calls the `uploadReport` function to upload the report to Google.

# Function: uploadReport
The `uploadReport` function is responsible for uploading a zipped folder to Google Drive. This function performs the following steps:
1. Initialize the timer to keep track of how long the operation takes.
2. Compress the folder specified by the `outputPath` into a .zip file.
3. Read the .zip file. If there's an error during this process, it logs the error and halts the execution of the program.
4. Upload the zipped file to Google Drive and log the process. If there's an error during the upload, it logs the error and halts the execution.
5. Once the upload is successful, it logs the file details including the name, web view link, file size, and the total time taken for the upload process.

# Function: googleAppsScriptAudit
The `googleAppsScriptAudit` function performs an audit operation over all the Google Apps Script projects in a Google domain. It's part of a larger system for Google Workspace administration. It retrieves and organizes detailed information about each project, including the project's name, ID, number, and service accounts.
1. The function initializes by creating a new `AppsScriptAPI` instance. This instance facilitates interactions with the Google Apps Script API.
2. It retrieves all the Google Apps Script projects in the Google Workspace domain by calling the `QueryProjects` method.
3. The function then loops over each project and, using goroutines for concurrency, fetches the project's name, ID, number, and service accounts.
4. This information is then collected into a CSV format, with each row corresponding to a project and containing the project's name, ID, number, and service accounts.
5. Once all projects have been processed, the function creates a new CSV file named `projects.csv` and writes all the collected information into it.
6. Finally, it calls the `uploadReport` function to upload the report to Google.