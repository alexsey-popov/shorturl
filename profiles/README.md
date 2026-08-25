# Что было сделано

> Для профилирования был взят код только одного пакета.
> 
> Описываю все свои действия пошагово т.к. занимаюсь профилированием впервые.
> 
> Не судите меня строго)

Под профилирование был выбран пакет `repository/inmemory` т.к. он используется в 2-х из 3-х видов репозиториев + с ним легче делать выводы о причинах потребления памяти.

Файл `base.pprof` был получен через запуск бенчмарков:
```bash
cd internal/repository/inmemory
go test -bench . -memprofile=base.pprof

goos: darwin
goarch: arm64
pkg: github.com/alexsey-popov/shorturl/internal/repository/inmemory
cpu: Apple M4
BenchmarkInMemory_Set-10                        79482478                13.81 ns/op
BenchmarkInMemory_Get-10                        135826382                9.067 ns/op
BenchmarkInMemory_FindFromOriginal-10           45807427                25.04 ns/op
BenchmarkInMemory_FindFromUserID-10             14037564                84.59 ns/op
PASS
ok      github.com/alexsey-popov/shorturl/internal/repository/inmemory  6.497s
```

Далее c помощью команды `go tool pprof base.pprof` я перешел в консольный режим и с помощью команды `top` смог получить сведения по 10 самым ресурсоёмким операциям.
```bash
(pprof) top
Showing nodes accounting for 3510.22MB, 99.63% of 3523.32MB total
Dropped 45 nodes (cum <= 17.62MB)
Showing top 10 nodes out of 12
      flat  flat%   sum%        cum   cum%
 3273.48MB 92.91% 92.91%  3273.48MB 92.91%  github.com/alexsey-popov/shorturl/internal/repository/inmemory.(*InMemory).FindFromUserID
  207.50MB  5.89% 98.80%   207.50MB  5.89%  github.com/alexsey-popov/shorturl/internal/repository/inmemory.(*InMemory).String
   19.62MB  0.56% 99.36%    19.62MB  0.56%  github.com/alexsey-popov/shorturl/internal/repository/inmemory..MarshalJSON.Collect[go.shape.struct { UUID string "json:\"uuid\""; Prefix string "json:\"short_url\""; OriginalURL string "json:\"original_url\""; UserID string "json:\"user_id\""; IsDeleted bool "json:\"is_deleted\"" }].AppendSeq[go.shape.[]go.shape.struct { UUID string "json:\"uuid\""; Prefix string "json:\"short_url\""; OriginalURL string "json:\"original_url\""; UserID string "json:\"user_id\""; IsDeleted bool "json:\"is_deleted\"" },go.shape.struct { UUID string "json:\"uuid\""; Prefix string "json:\"short_url\""; OriginalURL string "json:\"original_url\""; UserID string "json:\"user_id\""; IsDeleted bool "json:\"is_deleted\"" }]-range1 (inline)
    9.62MB  0.27% 99.63%    20.21MB  0.57%  encoding/json.Marshal
         0     0% 99.63%    39.83MB  1.13%  github.com/alexsey-popov/shorturl/internal/repository/inmemory.(*InMemory).MarshalJSON
         0     0% 99.63%    19.62MB  0.56%  github.com/alexsey-popov/shorturl/internal/repository/inmemory..MarshalJSON.Values[go.shape.map[string]github.com/alexsey-popov/shorturl/internal/model.URL,go.shape.string,go.shape.struct { UUID string "json:\"uuid\""; Prefix string "json:\"short_url\""; OriginalURL string "json:\"original_url\""; UserID string "json:\"user_id\""; IsDeleted bool "json:\"is_deleted\"" }].func1 (inline)
         0     0% 99.63%  3256.37MB 92.42%  github.com/alexsey-popov/shorturl/internal/repository/inmemory.BenchmarkInMemory_FindFromUserID
         0     0% 99.63%   264.43MB  7.51%  github.com/alexsey-popov/shorturl/internal/repository/inmemory.TestConcurrentAccess.func1
         0     0% 99.63%    19.62MB  0.56%  slices.AppendSeq[go.shape.[]go.shape.struct { UUID string "json:\"uuid\""; Prefix string "json:\"short_url\""; OriginalURL string "json:\"original_url\""; UserID string "json:\"user_id\""; IsDeleted bool "json:\"is_deleted\"" },go.shape.struct { UUID string "json:\"uuid\""; Prefix string "json:\"short_url\""; OriginalURL string "json:\"original_url\""; UserID string "json:\"user_id\""; IsDeleted bool "json:\"is_deleted\"" }] (inline)
         0     0% 99.63%    19.62MB  0.56%  slices.Collect[go.shape.struct { UUID string "json:\"uuid\""; Prefix string "json:\"short_url\""; OriginalURL string "json:\"original_url\""; UserID string "json:\"user_id\""; IsDeleted bool "json:\"is_deleted\"" }] (inline)
```

Затем я запросил сведения по операции `FindFromUserID` с привязкой к коду, чтобы понять в каком месте происходит наибольший расход памяти.
```bash
(pprof) list FindFromUserID 
Total: 3.44GB
ROUTINE ======================== github.com/alexsey-popov/shorturl/internal/repository/inmemory.(*InMemory).FindFromUserID in /Users/alexsey/GolandProjects/shorturl/internal/repository/inmemory/inmemory.go
    3.20GB     3.20GB (flat, cum) 92.91% of Total
         .          .    152:func (rep *InMemory) FindFromUserID(userID string) (urls []model.URL, err error) {
         .          .    153:   // Защищаем map от одновременного чтения из разных горутин
         .          .    154:   rep.mu.RLock()
         .          .    155:   defer rep.mu.RUnlock()
         .          .    156:
         .          .    157:   for _, item := range rep.data {
         .          .    158:           if item.UserID == userID {
         .          .    159:
    3.20GB     3.20GB    160:                   urls = append(urls, item)
         .          .    161:           }
         .          .    162:   }
         .          .    163:
         .          .    164:   return urls, nil
         .          .    165:}
ROUTINE ======================== github.com/alexsey-popov/shorturl/internal/repository/inmemory.BenchmarkInMemory_FindFromUserID in /Users/alexsey/GolandProjects/shorturl/internal/repository/inmemory/inmemory_test.go
         0     3.18GB (flat, cum) 92.42% of Total
         .          .    273:func BenchmarkInMemory_FindFromUserID(b *testing.B) {
         .          .    274:   rep := New()
         .          .    275:   urls := []model.URL{
         .          .    276:           {Prefix: "pref1", OriginalURL: "https://example.com/1", UserID: "user-1"},
         .          .    277:           {Prefix: "pref2", OriginalURL: "https://example.com/2", UserID: "user-1"},
         .          .    278:   }
         .          .    279:   _ = rep.SetMany(urls)
         .          .    280:   b.ResetTimer()
         .          .    281:   for i := 0; i < b.N; i++ {
         .     3.18GB    282:           _, _ = rep.FindFromUserID("user-1")
         .          .    283:   }
         .          .    284:}
```

Исходя из предоставленной информации видно, что вся память уходит на операцию append. Т.к. размер слайса у меня не задан, получается, что при многократной записи в слайс периодически происходят аллокации памяти в моменты, когда слайс достигает автоматически заданного предела (происходит выделение памяти под новый слайс большего размера и копирование всех предыдущих значений). 

Чтобы решить эту проблему я решил пробегать цикл дважды. На первой итерации получать количество элементов, а на второй записывать их.
> Скорее всего это далеко не самое элегантное решение, но другие инкременты тоже нужно кому-то делать)

После внесения изменений в метод `FindFromUserID` я повторил профилирование.

```bash
go test -bench . -memprofile=result.pprof

goos: darwin
goarch: arm64
pkg: github.com/alexsey-popov/shorturl/internal/repository/inmemory
cpu: Apple M4
BenchmarkInMemory_Set-10                        79785684                13.63 ns/op
BenchmarkInMemory_Get-10                        137020940                8.771 ns/op
BenchmarkInMemory_FindFromOriginal-10           50524986                24.16 ns/op
BenchmarkInMemory_FindFromUserID-10             14165998                83.44 ns/op
PASS
ok      github.com/alexsey-popov/shorturl/internal/repository/inmemory  6.505s

go tool pprof result.pprof     

(pprof) top
Showing nodes accounting for 2287.59MB, 99.58% of 2297.16MB total
Dropped 31 nodes (cum <= 11.49MB)
Showing top 10 nodes out of 12
      flat  flat%   sum%        cum   cum%
 2054.36MB 89.43% 89.43%  2054.36MB 89.43%  github.com/alexsey-popov/shorturl/internal/repository/inmemory.(*InMemory).FindFromUserID
  203.50MB  8.86% 98.29%   203.50MB  8.86%  github.com/alexsey-popov/shorturl/internal/repository/inmemory.(*InMemory).String
   18.61MB  0.81% 99.10%    18.61MB  0.81%  github.com/alexsey-popov/shorturl/internal/repository/inmemory..MarshalJSON.Collect[go.shape.struct { UUID string "json:\"uuid\""; Prefix string "json:\"short_url\""; OriginalURL string "json:\"original_url\""; UserID string "json:\"user_id\""; IsDeleted bool "json:\"is_deleted\"" }].AppendSeq[go.shape.[]go.shape.struct { UUID string "json:\"uuid\""; Prefix string "json:\"short_url\""; OriginalURL string "json:\"original_url\""; UserID string "json:\"user_id\""; IsDeleted bool "json:\"is_deleted\"" },go.shape.struct { UUID string "json:\"uuid\""; Prefix string "json:\"short_url\""; OriginalURL string "json:\"original_url\""; UserID string "json:\"user_id\""; IsDeleted bool "json:\"is_deleted\"" }]-range1 (inline)
   10.63MB  0.46% 99.56%    17.69MB  0.77%  encoding/json.Marshal
    0.50MB 0.022% 99.58%   249.38MB 10.86%  github.com/alexsey-popov/shorturl/internal/repository/inmemory.TestConcurrentAccess.func1
         0     0% 99.58%    36.30MB  1.58%  github.com/alexsey-popov/shorturl/internal/repository/inmemory.(*InMemory).MarshalJSON
         0     0% 99.58%    18.61MB  0.81%  github.com/alexsey-popov/shorturl/internal/repository/inmemory..MarshalJSON.Values[go.shape.map[string]github.com/alexsey-popov/shorturl/internal/model.URL,go.shape.string,go.shape.struct { UUID string "json:\"uuid\""; Prefix string "json:\"short_url\""; OriginalURL string "json:\"original_url\""; UserID string "json:\"user_id\""; IsDeleted bool "json:\"is_deleted\"" }].func1 (inline)
         0     0% 99.58%  2045.28MB 89.04%  github.com/alexsey-popov/shorturl/internal/repository/inmemory.BenchmarkInMemory_FindFromUserID
         0     0% 99.58%    18.61MB  0.81%  slices.AppendSeq[go.shape.[]go.shape.struct { UUID string "json:\"uuid\""; Prefix string "json:\"short_url\""; OriginalURL string "json:\"original_url\""; UserID string "json:\"user_id\""; IsDeleted bool "json:\"is_deleted\"" },go.shape.struct { UUID string "json:\"uuid\""; Prefix string "json:\"short_url\""; OriginalURL string "json:\"original_url\""; UserID string "json:\"user_id\""; IsDeleted bool "json:\"is_deleted\"" }] (inline)
         0     0% 99.58%    18.61MB  0.81%  slices.Collect[go.shape.struct { UUID string "json:\"uuid\""; Prefix string "json:\"short_url\""; OriginalURL string "json:\"original_url\""; UserID string "json:\"user_id\""; IsDeleted bool "json:\"is_deleted\"" }] (inline)


(pprof) list FindFromUserID 
Total: 2.24GB
ROUTINE ======================== github.com/alexsey-popov/shorturl/internal/repository/inmemory.(*InMemory).FindFromUserID in /Users/alexsey/GolandProjects/shorturl/internal/repository/inmemory/inmemory.go
    2.01GB     2.01GB (flat, cum) 89.43% of Total
         .          .    152:func (rep *InMemory) FindFromUserID(userID string) (urls []model.URL, err error) {
         .          .    153:   // Защищаем map от одновременного чтения из разных горутин
         .          .    154:   rep.mu.RLock()
         .          .    155:   defer rep.mu.RUnlock()
         .          .    156:
         .          .    157:   count := 0
         .          .    158:   for _, item := range rep.data {
         .          .    159:           if item.UserID == userID {
         .          .    160:                   count++
         .          .    161:           }
         .          .    162:   }
         .          .    163:
         .          .    164:   if count == 0 {
         .          .    165:           return nil, nil
         .          .    166:   }
         .          .    167:
    2.01GB     2.01GB    168:   urls = make([]model.URL, 0, count)
         .          .    169:   for _, item := range rep.data {
         .          .    170:           if item.UserID == userID {
         .          .    171:                   urls = append(urls, item)
         .          .    172:           }
         .          .    173:   }
ROUTINE ======================== github.com/alexsey-popov/shorturl/internal/repository/inmemory.BenchmarkInMemory_FindFromUserID in /Users/alexsey/GolandProjects/shorturl/internal/repository/inmemory/inmemory_test.go
         0        2GB (flat, cum) 89.04% of Total
         .          .    273:func BenchmarkInMemory_FindFromUserID(b *testing.B) {
         .          .    274:   rep := New()
         .          .    275:   urls := []model.URL{
         .          .    276:           {Prefix: "pref1", OriginalURL: "https://example.com/1", UserID: "user-1"},
         .          .    277:           {Prefix: "pref2", OriginalURL: "https://example.com/2", UserID: "user-1"},
         .          .    278:   }
         .          .    279:   _ = rep.SetMany(urls)
         .          .    280:   b.ResetTimer()
         .          .    281:   for i := 0; i < b.N; i++ {
         .        2GB    282:           _, _ = rep.FindFromUserID("user-1")
         .          .    283:   }
         .          .    284:}
```

Для того, чтобы сравнить результаты я выполнил команду `go tool pprof -top -diff_base=./base.pprof ./result.pprof `, которая выдала следующий результат:
```bash
File: inmemory.test
Type: alloc_space
Time: 2026-08-25 19:02:43 MSK
Showing nodes accounting for -1218.62MB, 34.59% of 3523.32MB total
Dropped 57 nodes (cum <= 17.62MB)
      flat  flat%   sum%        cum   cum%
-1219.12MB 34.60% 34.60% -1219.12MB 34.60%  github.com/alexsey-popov/shorturl/internal/repository/inmemory.(*InMemory).FindFromUserID
    0.50MB 0.014% 34.59%   -15.05MB  0.43%  github.com/alexsey-popov/shorturl/internal/repository/inmemory.TestConcurrentAccess.func1
         0     0% 34.59% -1211.09MB 34.37%  github.com/alexsey-popov/shorturl/internal/repository/inmemory.BenchmarkInMemory_FindFromUserID
         0     0% 34.59% -1211.09MB 34.37%  testing.(*B).launch
         0     0% 34.59% -1211.61MB 34.39%  testing.(*B).runN
```

Вывод: Через однократную фиксацию длины слайса получилось снизить расход памяти на 34.6%.

> Однако данный вывод справедлив только для конкретного соотношения ссылок в репозитории к пользователю 