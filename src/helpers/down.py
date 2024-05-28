import os

def download_image(image_url, file_name, referer, directory_path, pbar):

    unscramble = False
    if 'clel' in image_url:
        unscramble = True

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

        if unscramble is True:
            scrambled_image = file_check_path + '_scrambled'
        else:
            scrambled_image = file_check_path

        file = open(scrambled_image, 'wb')
        file.write(image_content)
        file.close()

        if unscramble is True:
            self.unscramble_image(scrambled_image, file_check_path)
            os.remove(scrambled_image)