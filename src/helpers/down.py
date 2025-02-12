import os


def download_image(image_url, file_name, referer, directory_path, pbar):
    file_check_path = str(directory_path) + os.sep + str(file_name)
    if os.path.isfile(file_check_path):
        pbar.write('[Comic-dl] File Exist! Skipping : %s\n' % file_name)
        pass

    if not os.path.isfile(file_check_path):
        headers = {
            'User-Agent':
                'Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/56.0.2924.87 Safari/537.36',
            'Accept-Encoding': 'gzip, deflate',
            'Referer': referer
        }

        image_content = self.scraper.get(image_url, headers=headers).content

        with open(file_check_path, 'wb') as file:
            file.write(image_content)
            file.close()
