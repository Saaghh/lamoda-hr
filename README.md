Это тестовое задание от lamoda, сделанное Семиным Сергеем.

Результатом этого тестового задания является код сервиса для работы с Резервацией товара на складе.
Код покрыт тестами на ~79%

Для запуска сервиса требуется докер и golang версии 1.22

Для запуска сервиса и поднятия окружения в докере используйте команду `$ make up` 
При первом запуске контейнера с постгресом будут вставлены данные из .sql файлов находящихся в папке `initdb`
Основной сервис запускается с `restart: always`, чтобы дождаться запуска контейнера постгреса.

Для запуска тестов используйте `$ make test`

Если первой командой использовать `$ make test`, она поднимет необходимое окружение, но при первом запуске упадет, 
Связано это с тем, что докер контейнер с потсгресом не успевает подняться до запуска тестов.

Рекомендуется сначала использовать `$ make up`, дождаться поднятия контейнера и после запускать `$ make test`

Тесты самостоятельно добавляют необходимые данные в базу данных и удаляют их при завершении. 
То есть для них подойдет пустая база данных. 

Также тесты поднимают свою копию сервиса, а не используют поднятую в контейнере.
Сделано это для ускорения и упрощения отладки и возможности проверки покрытия.

Документация к api находится в папке `api` в формате openapi 3.0.3
Коллекцию postman можно собрать, импортировав этот файл в приложение postman. 

Код оформатирован с ипользованием `gofumpt`, `gci` и `golangci-lint`. 
Первые два - более строгие аналоги того, что представлено в требования

Если есть вопросы, буду рад обсудить их более подробно!
