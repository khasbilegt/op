import json
import re
import requests
from pathlib import Path
from bs4 import BeautifulSoup

ROOT_URL = "https://mangapill.com/manga/3258/one-piece-digital-colored-comics"

path = Path.cwd() / "links.txt"
data = Path.cwd() / "data.json"
page = requests.get(ROOT_URL).text
soup = BeautifulSoup(page, "html.parser")
links = soup.find_all(
    "a", class_="py-1 px-2 border border-color-border-secondary rounded text-sm"
)

# for link in reversed(links):
#     if not path.exists():
#         path.touch()
#     with path.open(mode="a") as f:
#         f.write(f"\n{ROOT_URL}{link['href']}")

CHAPTER_LINKS = [f"https://mangapill.com{link['href']}" for link in reversed(links)]
FINAL_DATA = {}
chapter_links_len = len(CHAPTER_LINKS)

for index, chapter_link in enumerate(CHAPTER_LINKS, start=1):
    page = requests.get(chapter_link).text
    soup = BeautifulSoup(page, "html.parser")
    page_titles = soup.find_all(
        "div",
        class_="px-2 py-1 border-b border-color-border-primary bg-color-bg-primary uppercase text-sm text-color-text-secondary text-center",
        string=re.compile(r"^page"),
    )
    images = soup.find_all("img", class_="lazy js-page")

    image_links = list(
        image["data-src"]
        for image in images
        if (
            image.get("data-src")
            and "https://cdn.readdetectiveconan.com/" in image["data-src"]
        )
    )
    titles = list(f"{title.string.split(' ')[-1]}.jpg" for title in page_titles)

    if len(titles) == len(image_links):
        FINAL_DATA.update(
            {
                chapter_link.split("/")[-1]
                .replace("-", " ")
                .title(): dict(zip(titles, image_links))
            }
        )
        print(
            f"Chapter No.{index:3} ({len(image_links):2} images) out of {chapter_links_len} --- {(index * 100) / chapter_links_len:4.1f}%"
        )
    else:
        print(
            f"Chapter No.{index:3}: pageCount={len(page_titles)}, imageCount={len(image_links)}"
        )

if not data.exists():
    data.touch()
with data.open(mode="w") as f:
    json.dump(FINAL_DATA, f)
