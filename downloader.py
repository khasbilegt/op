import concurrent.futures
import json
import requests
import shutil
from pathlib import Path


PATH = Path.cwd() / "data.json"
LINKS = {}

if PATH.exists() and PATH.is_file():
    with open(PATH, mode="r") as f:
        LINKS = json.load(f)


def downloadChapter(input):
    chapterName, images = input
    image_count = len(images)
    folder = Path.cwd() / "One Piece" / chapterName

    if not folder.exists():
        folder.mkdir(parents=True)

    for filename, url in images.items():
        response = requests.get(url, stream=True)
        if response.status_code == 200:
            response.raw.decode_content = True

            with open(f"{folder}/{filename}", "wb") as f:
                shutil.copyfileobj(response.raw, f)

            print(
                f"✅ Saved image no. {filename.strip('.jpg'):2}/{image_count} of \"Chapter {chapterName.split(' ')[-1]}\""
            )
        else:
            print(f"🚫 Wrong image url: {url}")


def main():
    with concurrent.futures.ProcessPoolExecutor(max_workers=16) as executor:
        executor.map(downloadChapter, LINKS.items())


if __name__ == "__main__":
    main()
