# <p style="text-align: center;">Dokumentace k semestrální práci z předmětu KIV/ZOS <br><br>Téma práce: Zjednodušený souborový systéme založený na pseudoFAT</p>

> Autor: Milan Vlachovský

## Obsah

- [1. Úvod](#1-úvod)
- [2. Popis programu](#2-popis-hry)
- [3. Architektura systému](#4-architektura-systému)
- [4. Návod na zprovoznění](#5-návod-na-zprovoznění)
- [5. Struktura projektu](#6-struktura-projektu)
- [6. Popis implementace](#7-popis-implementace)
- [7. Závěr](#8-závěr)

## 1. Úvod

<style>body {text-align: justify}</style>

Cílem tohoto projektu bylo vytvořit elementární hru pro více hráčů používající síťovou komunikaci se serverovou částí v nízkoúrovňovém programovacím jazyce a klientskou částí v programovacím libovolném jazyce. Autor projektu zvolil vlastní variantu hry Lodě s upravenými pravidly nazvanou *&bdquo;Inverse Battleships&ldquo;*. Stejně jako její předloha je hra určena pro 2 hráče, přičemž se hráčí střídají v tazích.
&nbsp;&nbsp;&nbsp;&nbsp;Byl zvolen protokol na bázi TCP. Jako programovací jazyk serveru byl zvolen jazyk Go, díky své rychlosti a nízkoúrovňovému přístupu k síťové komunikaci. Klientská část byla implementována v jazyce Python s využitím knihovny [pygame](https://www.pygame.org/news) pro správu vykreslování grafického prostředí hry.

## 2. Popis programu

Hra *Inverse Battleships* je variantou klasické hry Lodě, ve které se hráči snaží najít a zničit všechny lodě protivníka. V této variantě hráči sdílí jedno pole 9x9 a každý dostane na začátku přiřazenou jednu loď. Následně se hráči střídají v tazích, kdy každý hráč se může pokusit vykonat akci na prázdné políčko. Mohou nastat tři situace:

- Hráč zkusí akci na prázdné políčko, ve kterém se nachází nikým nezískaná loď. V tomto případě hráč loď získává a získává body.  

- Hráč zkusí akci na prázdné políčko, ve kterém se nachází protivníkova loď. V tomto případě protivník o loď přichází, je zničena, hráč získává body a protivník ztrácí body.

- Hráč zkusí akci na políčko, na kterém se nic nenachází. V tomto případě hráč nezískává nic.

Hra končí, když jeden z hráčů ztratí všechny lodě. Vítězem je přeživší hráč. Body jsou pouze pro statistické účely a nemají vliv na průběh hry.

## 3. Architektura systému

Systém je rozdělen na dvě části: serverovou a klientskou. Serverová část je napsána v jazyce Go a klientská část v jazyce Python s využitím knihovny *pygame*. Serverová část je zodpovědná za správu hry, komunikaci s klienty a validaci zpráv. Klientská část je zodpovědná za zobrazení grafického rozhraní, zpracování vstupů od uživatele a komunikaci se serverem. 
&nbsp;&nbsp;&nbsp;&nbsp;V obou částech je síťová komunikace zajištěna na nízké úrovni pomocí socketů (v Pythonu pomocí knihovny *socket* a v Go pomocí knihovny *net*). Jedná se o tahovou hru, kde se hráči střídají v tazích. Proto byl jako protokol zvolen TCP, který je vhodný pro tento typ hry.

### Požadavky

Projekt byl vyvíjen s použitím následujících technologií:

- Python 3.12
  - pygame 2.6.0
  - pydantic 2.8.2
  - typing-extensions 4.12.2
  - termcolor 2.5.0
  - pyinstaller 6.11.1
- Go 1.23

> Za použití zmiňovaných technologií by měl být projekt bez problémů spustitelný. Spouštění na starších verzích nebylo testováno a nemusí fungovat správně.

## 4. Návod na zprovoznění

Pro sestavení celého projektu byly vytvořeny soubory *Makefile* a *Makefile.win*, které obsahují instrukce pro sestavení projektu na Unixových a Windows OS. Pro sestavení projektu na Unixových OS stačí spustit příkaz:

```bash
make
```

a pro Windows OS stačí spustit příkaz:

```cmd
make -f Makefile.win
```

Předpokládá se, že je nainstalován program `make`; na Windows je možné použít například [make z chocolatey](https://community.chocolatey.org/packages/make), či jiné alternativy. 
&nbsp;&nbsp;&nbsp;&nbsp;Skript sestaví spustitelné soubory ve složce *client/bin/* pro klientskou část projektu a ve složce *server/bin/* pro serverovou část projektu. Spustitelné soubory jsou pojmenovány *client* a *server*, případně na Windows *client.exe* a *server.exe*. Stačí pouze z kořenové složky projektu na Unix OS spustit příkaz:

```bash
make
```

Nebo na Windows OS:

```cmd
make -f Makefile.win
```

> Jelikož je klientská část implementována v jazyce Python, je možné ji spustit i bez sestavení. Stačí spustit soubor *client/src/main.py* v Python virtuálním prostředí s nainstalovanými závislostmi ze souboru *requirements.txt*. Spustitelné soubory pro klientskou část byly vytvořeny pomocí knihovny *[pyinstaller](https://pyinstaller.org/en/stable/)* a jejich úspěšnost překladu bývá závislá na operačním systému a verzi Pythonu.

> Na základě standardu [PEP 394](https://peps.python.org/pep-0394/) počítají soubory *Makefile* a *Makefile.win* s tím, že Python rozkaz pod Unixem je `python3` a pod Windows je `python`. V případě odlišného nastavení je nutné soubory upravit.

## 5. Struktura projektu

Projekt je rozdělen následovně:

- *Kořenová složka* &mdash; Obsahuje soubory pro sestavení projektu, složky *client* a *server*, složku *docs* s dokumentací a soubor *requirements.txt* s definicemi Python závislostí.
  - *client/* &mdash; Obsahuje celou klientskou část projektu včetně konfiguračních souborů, použitých textových a obrazových zdrojů a programátorské referenční dokumentace.
  - *server/* &mdash; Obsahuje celou serverovou část projektu včetně konfiguračních souborů a programátorské referenční dokumentace.

### Kořenová složka

<!-- strom: -->
<!-- .
├── ./Makefile
├── ./Makefile.win
├── ./ZOS2024_SP.pdf
├── ./ZOS2024_SP_zadani.txt
├── ./bin
│   ├── ./bin/img.gif
│   ├── ./bin/img2.gif
│   ├── ./bin/main.go
│   ├── ./bin/main.txt
│   ├── ./bin/myfs
│   ├── ./bin/myfs.dat
│   ├── ./bin/myfs.exe
│   ├── ./bin/s1.txt
│   ├── ./bin/test.cmds
│   ├── ./bin/testh.cmds
│   └── ./bin/testr.cmds
├── ./compute_fat_count.py
├── ./docs
│   └── ./docs/doc.md
├── ./fat_count.csv
├── ./fat_hint.webp
└── ./src
    ├── ./src/arg_parser
    │   └── ./src/arg_parser/arg_parser.go
    ├── ./src/cmd
    │   ├── ./src/cmd/command.go
    │   ├── ./src/cmd/command_executor.go
    │   ├── ./src/cmd/command_parser.go
    │   └── ./src/cmd/command_validator.go
    ├── ./src/consts
    │   ├── ./src/consts/cmds.go
    │   ├── ./src/consts/exit_codes.go
    │   ├── ./src/consts/fat_flags.go
    │   ├── ./src/consts/formats.go
    │   ├── ./src/consts/limits.go
    │   └── ./src/consts/msg.go
    ├── ./src/custom_errors
    │   └── ./src/custom_errors/errors.go
    ├── ./src/go.mod
    ├── ./src/logging
    │   └── ./src/logging/logging.go
    ├── ./src/main.go
    ├── ./src/myfs.dat
    ├── ./src/pseudo_fat
    │   └── ./src/pseudo_fat/structures.go
    ├── ./src/test.cmds
    ├── ./src/testh.cmds
    ├── ./src/testr.cmds
    ├── ./src/tmp
    └── ./src/utils
        ├── ./src/utils/data_transform.go
        ├── ./src/utils/loader.go
        ├── ./src/utils/path.go
        ├── ./src/utils/pretty_print.go
        └── ./src/utils/pseudo_fat_fs_operations.go -->

- *Makefile* &mdash; Soubor pro sestavení projektu na Unix OS.

- *Makefile.win* &mdash; Soubor pro sestavení projektu na Windows OS.


- *docs/* &mdash; Složka obsahující dokumentaci.
  - *docs/doc.md* a *docs/doc.pdf* &mdash; Tento dokument ve formátu Markdown a PDF.
  - *docs/client_ref.html* &mdash; Odkaz na dokumentaci klientské části.
  - *docs/server_ref.html* &mdash; Odkaz na dokumentaci serverové části.

## 6. Popis implementace

Do popisu implementace budou převážně zahrnuty jen ty nejdůležitější části kódu potřebné pro pochopení principu fungování aplikace. Pro podrobnější informace je možné využít programátorskou referenční dokumentaci, která je dostupná v *docs/client_ref.html* a *docs/server_ref.html* (odkazují do *client/docs/* a *server/docs/*).

## 7. Závěr

