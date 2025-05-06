# PR4

## Особенности реализации
1) Серверная часть:
  1.Генерация случайного 4-символьного кода
  2.Поддержка 2-4 игроков одновременно
  3.Асинхронная обработка подключений
  4.Механизм проверки догадок с подсчетом черных и белых маркеров
  5.Сохранение результатов каждого раунда в XML-файл
  6.Автоматический запуск новых раундов

2) Клиентская часть:
  1.Подключение к серверу
  2.Ввод имени игрока
  3.Отправка догадок и получение результатов
  4.Отображение количества черных и белых маркеров

3) Обмен данными:
  1.Использование XML для сериализации сообщений
  2.TCP/IP для сетевого взаимодействия
  3.Асинхронные операции ввода-вывода

4) Обработка ошибок:
  1.Проверка корректности ввода
  2.Обработка сетевых ошибок
  3.Защита от переполнения сервера

## Серверная часть
```
using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using System.Net;
using System.Net.Sockets;
using System.Text;
using System.Threading.Tasks;
using System.Xml.Serialization;

namespace CodeMasterServer
{
    public class GameRound
    {
        public DateTime StartTime { get; set; }
        public DateTime? EndTime { get; set; }
        public string SecretCode { get; set; }
        public List<PlayerResult> PlayerResults { get; set; } = new List<PlayerResult>();
        public string Winner { get; set; }
    }

    public class PlayerResult
    {
        public string PlayerName { get; set; }
        public int Attempts { get; set; }
    }

    public class GameResponse
    {
        public int BlackMarkers { get; set; }
        public int WhiteMarkers { get; set; }
        public bool IsCorrect { get; set; }
        public string Message { get; set; }
    }

    class Program
    {
        private const int Port = 8888;
        private const int CodeLength = 4;
        private const int MaxAttempts = 10;
        private const int MaxPlayers = 4;
        private const int MinPlayers = 2;
        private static readonly string ResultsFilePath = "game_results.xml";
        private static readonly List<GameRound> gameHistory = new List<GameRound>();
        private static readonly object gameLock = new object();
        private static GameRound currentRound;
        private static List<TcpClient> connectedClients = new List<TcpClient>();
        private static Dictionary<TcpClient, string> clientNames = new Dictionary<TcpClient, string>();
        private static Dictionary<TcpClient, int> clientAttempts = new Dictionary<TcpClient, int>();

        static async Task Main(string[] args)
        {
            Console.WriteLine("Сервер игры 'Код-Мастер' запущен");
            var listener = new TcpListener(IPAddress.Any, Port);
            listener.Start();

            _ = Task.Run(() => ManageGameRounds());

            while (true)
            {
                var client = await listener.AcceptTcpClientAsync();
                _ = HandleClientAsync(client);
            }
        }

        private static async Task ManageGameRounds()
        {
            while (true)
            {
                // Ждем минимального количества игроков
                while (connectedClients.Count < MinPlayers)
                {
                    await Task.Delay(1000);
                }

                StartNewRound();

                // Ожидаем завершения раунда
                while (currentRound != null && currentRound.EndTime == null)
                {
                    await Task.Delay(1000);
                }
            }
        }

        private static void StartNewRound()
        {
            lock (gameLock)
            {
                currentRound = new GameRound
                {
                    StartTime = DateTime.Now,
                    SecretCode = GenerateSecretCode(),
                    PlayerResults = new List<PlayerResult>()
                };

                foreach (var client in connectedClients)
                {
                    clientAttempts[client] = 0;
                }

                Console.WriteLine($"Начался новый раунд. Секретный код: {currentRound.SecretCode}");
                BroadcastMessage("Новый раунд начался! У вас есть 10 попыток угадать 4-значный код (A-Z, 0-9)");
            }
        }

        private static string GenerateSecretCode()
        {
            var random = new Random();
            const string chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789";
            return new string(Enumerable.Repeat(chars, CodeLength)
                .Select(s => s[random.Next(s.Length)]).ToArray());
        }

        private static async Task HandleClientAsync(TcpClient client)
        {
            try
            {
                string playerName = null;

                using (var stream = client.GetStream())
                using (var reader = new StreamReader(stream, Encoding.UTF8))
                using (var writer = new StreamWriter(stream, Encoding.UTF8) { AutoFlush = true })
                {
                    // Получаем имя игрока
                    await writer.WriteLineAsync("Введите ваше имя:");
                    playerName = await reader.ReadLineAsync();

                    lock (gameLock)
                    {
                        if (connectedClients.Count >= MaxPlayers)
                        {
                            await writer.WriteLineAsync("Сервер переполнен. Попробуйте подключиться позже.");
                            return;
                        }

                        connectedClients.Add(client);
                        clientNames[client] = playerName;
                        clientAttempts[client] = 0;
                    }

                    await writer.WriteLineAsync($"Добро пожаловать, {playerName}! Ожидайте начала раунда...");

                    // Основной игровой цикл
                    while (true)
                    {
                        if (currentRound == null || currentRound.EndTime != null)
                        {
                            await Task.Delay(1000);
                            continue;
                        }

                        await writer.WriteLineAsync("Введите вашу догадку (4 символа A-Z или 0-9):");
                        var guess = await reader.ReadLineAsync();

                        if (string.IsNullOrEmpty(guess) || guess.Length != CodeLength)
                        {
                            await writer.WriteLineAsync("Неверный формат. Введите 4 символа (A-Z или 0-9)");
                            continue;
                        }

                        var response = ProcessGuess(guess.ToUpper(), client);
                        await writer.WriteLineAsync(SerializeResponse(response));

                        if (response.IsCorrect)
                        {
                            await writer.WriteLineAsync($"Поздравляем! Вы угадали код: {currentRound.SecretCode}");
                            break;
                        }

                        if (clientAttempts[client] >= MaxAttempts)
                        {
                            await writer.WriteLineAsync($"У вас закончились попытки. Секретный код был: {currentRound.SecretCode}");
                            break;
                        }
                    }
                }
            }
            catch (Exception ex)
            {
                Console.WriteLine($"Ошибка с клиентом: {ex.Message}");
            }
            finally
            {
                lock (gameLock)
                {
                    connectedClients.Remove(client);
                    clientNames.Remove(client);
                    clientAttempts.Remove(client);
                }
            }
        }

        private static GameResponse ProcessGuess(string guess, TcpClient client)
        {
            lock (gameLock)
            {
                if (currentRound == null || currentRound.EndTime != null)
                    return new GameResponse { Message = "Раунд не активен" };

                clientAttempts[client]++;

                var secretCode = currentRound.SecretCode;
                var blackMarkers = 0;
                var whiteMarkers = 0;
                var tempSecret = secretCode.ToCharArray();
                var tempGuess = guess.ToCharArray();

                // Проверка на черные маркеры (правильные символы на правильных позициях)
                for (int i = 0; i < CodeLength; i++)
                {
                    if (tempGuess[i] == tempSecret[i])
                    {
                        blackMarkers++;
                        tempSecret[i] = '_';
                        tempGuess[i] = '_';
                    }
                }

                // Проверка на белые маркеры (правильные символы на неправильных позициях)
                for (int i = 0; i < CodeLength; i++)
                {
                    if (tempGuess[i] == '_') continue;

                    for (int j = 0; j < CodeLength; j++)
                    {
                        if (tempSecret[j] == '_') continue;

                        if (tempGuess[i] == tempSecret[j])
                        {
                            whiteMarkers++;
                            tempSecret[j] = '_';
                            tempGuess[i] = '_';
                            break;
                        }
                    }
                }

                var isCorrect = blackMarkers == CodeLength;
                if (isCorrect || connectedClients.All(c => clientAttempts[c] >= MaxAttempts))
                {
                    EndRound(client, isCorrect);
                }

                return new GameResponse
                {
                    BlackMarkers = blackMarkers,
                    WhiteMarkers = whiteMarkers,
                    IsCorrect = isCorrect,
                    Message = $"Черные: {blackMarkers}, Белые: {whiteMarkers}"
                };
            }
        }

        private static void EndRound(TcpClient winnerClient, bool hasWinner)
        {
            currentRound.EndTime = DateTime.Now;

            if (hasWinner)
            {
                var winnerName = clientNames[winnerClient];
                currentRound.Winner = winnerName;
                BroadcastMessage($"Игрок {winnerName} угадал код {currentRound.SecretCode}!");
            }
            else
            {
                currentRound.Winner = "Нет победителя";
                BroadcastMessage($"Никто не угадал код. Секретный код был: {currentRound.SecretCode}");
            }

            // Сохраняем результаты игроков
            foreach (var client in connectedClients)
            {
                currentRound.PlayerResults.Add(new PlayerResult
                {
                    PlayerName = clientNames[client],
                    Attempts = clientAttempts[client]
                });
            }

            // Сохраняем раунд в историю
            gameHistory.Add(currentRound);
            SaveGameHistory();

            // Подготовка к новому раунду
            currentRound = null;
        }

        private static async void BroadcastMessage(string message)
        {
            foreach (var client in connectedClients.ToList())
            {
                try
                {
                    var stream = client.GetStream();
                    var writer = new StreamWriter(stream, Encoding.UTF8) { AutoFlush = true };
                    await writer.WriteLineAsync(message);
                }
                catch
                {
                    // Игрок отключился
                }
            }
        }

        private static void SaveGameHistory()
        {
            try
            {
                var serializer = new XmlSerializer(typeof(List<GameRound>));
                using (var writer = new StreamWriter(ResultsFilePath))
                {
                    serializer.Serialize(writer, gameHistory);
                }
            }
            catch (Exception ex)
            {
                Console.WriteLine($"Ошибка при сохранении истории: {ex.Message}");
            }
        }

        private static string SerializeResponse(GameResponse response)
        {
            var serializer = new XmlSerializer(typeof(GameResponse));
            using (var writer = new StringWriter())
            {
                serializer.Serialize(writer, response);
                return writer.ToString();
            }
        }
    }
}
```

## Клиентская часть
```
using System;
using System.IO;
using System.Net.Sockets;
using System.Text;
using System.Threading.Tasks;
using System.Xml.Serialization;

namespace CodeMasterClient
{
    public class GameResponse
    {
        public int BlackMarkers { get; set; }
        public int WhiteMarkers { get; set; }
        public bool IsCorrect { get; set; }
        public string Message { get; set; }
    }

    class Program
    {
        private const string ServerAddress = "127.0.0.1";
        private const int Port = 8888;

        static async Task Main(string[] args)
        {
            Console.WriteLine("Клиент игры 'Код-Мастер'");
            Console.WriteLine("Подключение к серверу...");

            try
            {
                using (var client = new TcpClient())
                {
                    await client.ConnectAsync(ServerAddress, Port);
                    Console.WriteLine("Подключено к серверу");

                    using (var stream = client.GetStream())
                    using (var reader = new StreamReader(stream, Encoding.UTF8))
                    using (var writer = new StreamWriter(stream, Encoding.UTF8) { AutoFlush = true })
                    {
                        // Получаем приветственное сообщение
                        Console.WriteLine(await reader.ReadLineAsync());

                        // Отправляем имя игрока
                        var playerName = Console.ReadLine();
                        await writer.WriteLineAsync(playerName);

                        // Основной игровой цикл
                        while (true)
                        {
                            var serverMessage = await reader.ReadLineAsync();
                            Console.WriteLine(serverMessage);

                            if (serverMessage.Contains("Поздравляем") || serverMessage.Contains("Секретный код был"))
                            {
                                break;
                            }

                            if (serverMessage.Contains("Введите вашу догадку"))
                            {
                                var guess = Console.ReadLine();
                                await writer.WriteLineAsync(guess);

                                var responseXml = await reader.ReadLineAsync();
                                var response = DeserializeResponse(responseXml);
                                Console.WriteLine(response.Message);
                            }
                        }
                    }
                }
            }
            catch (Exception ex)
            {
                Console.WriteLine($"Ошибка: {ex.Message}");
            }

            Console.WriteLine("Нажмите любую клавишу для выхода...");
            Console.ReadKey();
        }

        private static GameResponse DeserializeResponse(string xml)
        {
            var serializer = new XmlSerializer(typeof(GameResponse));
            using (var reader = new StringReader(xml))
            {
                return (GameResponse)serializer.Deserialize(reader);
            }
        }
    }
}
```
