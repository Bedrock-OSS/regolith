---
title: Installing
---

# Installing

## Windows Installation Using `winget`

::: info
Not every Windows computer has winget installed. If this is the case for your computer, you can install Regolith using the MSI file available on GitHub (see next section for instructions).
:::

To install the application "Regolith" using winget, follow these steps:

1. Open a command prompt or terminal window and enter the following command:

```
winget install Bedrock-OSS.regolith
```
This will search the winget repository for the package "Bedrock-OSS.regolith" and install it on your system.

2. If the installation is successful, you should see a message indicating that the package has been installed.

To update Regolith in the future, simply run the following command:

```
winget upgrade Bedrock-OSS.regolith
```

This will check for any available updates to the Regolith package and install them on your system.

## Windows Installation Using an `.msi` File

### Installation

Alternatively, you can install Regolith using the MSI file available on GitHub at the following link: https://github.com/Bedrock-OSS/regolith/releases/latest. The file will be named using the pattern `regolith-x.x.x.msi`, where `x.x.x` is the version number. To install Regolith using the MSI file, follow these steps:

Download the MSI file from the link above.

![](/installing/msi_download.png)

Run the MSI file to begin the installation process. Follow the prompts to complete the installation.

![](/installing/regolith_msi.png)

#### Updates

To update Regolith after installation, you can use the "regolith-update.ps1" PowerShell script that is included with the installation. To run the script, follow these steps:

1. Open a PowerShell window.
2. Run the following command:

```
regolith-update.ps1
```

This will check for any available updates to Regolith and install them on your system.

## Linux, Mac, and Windows (stand-alone)

Regolith can also be installed stand-alone. Simply install the correct zip for your operating system. For Windows, this is most likely `regolith_x.x.x_Windows_x86_64.zip`.

![](/installing/exe_download.png)

You may unzip this package, and place the `regolith.exe` file somewhere convenient. In stand-alone mode, you will need a copy of the regolith executable in every project that you intend to use Regolith with. Or, you can add the executable to your PATH environment variable.

## Checking Installation

After installing, Regolith can be used in any command-prompt by typing `regolith`. You should see something like this:

![](/installing/regolith_help.png)