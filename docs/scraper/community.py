import os, json, requests, shutil

class Filter:
    def __init__(self, name: str, author: str, url: str, lang: str, desc: str):
        self.name = name
        self.author = author
        self.url = url
        self.lang = lang
        self.desc = desc
    
    def toTable(self):
        return "| [" + self.name + "](" + self.url + ") | " + self.author + " | " + self.lang + " | " + self.desc + " |"

def comFilters():
    result = []
    root = os.getcwd()
    if not os.path.isdir("./tmp"):
        os.mkdir("./tmp")
    os.chdir("./tmp")
    tmp = os.getcwd()
    topicUrl = """https://api.github.com/search/repositories?q=topic:regolith-filter"""
    topicJson = requests.get(topicUrl).json()
    for repo in topicJson["items"]:
        fAuthor = repo["owner"]["login"]
        if fAuthor == "Bedrock-OSS": continue
        clone = repo["clone_url"]
        baseUrl = repo["svn_url"]
        repoFolder = repo["full_name"].replace("/", "_")
        os.system("git clone --depth=1 " + clone + " " + repoFolder)
        print("\n" + repoFolder + ":")
        os.chdir(repoFolder)
        for f in os.listdir("."):
            if f.startswith(".") and not os.path.isfile(f):
                os.system("rmdir /s /q " + f)
                continue
            if f.startswith(".") or os.path.isfile(f): continue
            fPath = "./" + f + "/"
            if not os.path.isfile(fPath + "filter.json"): continue
            fName = f
            fUrl = baseUrl + "/tree/" + repo["default_branch"] + "/" + fName
            fJson = json.load(open(fPath + "filter.json", "r"))
            fLang = fJson["filters"][0]["runWith"]
            fDesc = ""
            def md(path: str, readmeFile: str):
                result = ""
                m = open(path + readmeFile, "r", encoding='utf-8')
                try:
                    mStr = m.read()
                except:
                    print("repo (" + fAuthor + "_" + fName + "): has invalid readme file setting description to \"\"")
                    return result
                mLines = mStr.split("\n")
                for i in mLines:
                    if i.startswith("#"): continue
                    elif i == "": continue
                    else: 
                        result = i
                        break
                return result
            if os.path.isfile(fPath + "readme.md"): fDesc = md(fPath, "readme.md")
            elif os.path.isfile(fPath + "README.md"): fDesc = md(fPath, "README.md")
            result.append(Filter(fName, fAuthor, fUrl, fLang, fDesc))
            print("Added filter:" + fName + " to community filters")
        for d in os.listdir("."):
            if os.path.isdir(d): shutil.rmtree(d)
        os.chdir(tmp)
        shutil.rmtree(repoFolder)
    os.chdir(root)
    os.rmdir(tmp)
    return result

def filterTable(fList):
    tables = []
    for f in fList: tables.append(f.toTable())
    return "\n".join(tables)

def updateFilters(ftables, output: str):
    base = open("./community_base.md", "r").read()
    if os.path.isfile(output): os.remove(output)
    updated = open(output, "x")
    updated.write(base)
    updated.write("\n")
    updated.write(ftables)
    updated.close()
    print("Updated Community Filters")