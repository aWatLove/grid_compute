def generate(input_data, amount, start):
    """
    Генерирует маршруты, которые:
    1. Посещают ВСЕ города минимум 1 раз
    2. Возвращаются в стартовый город
    3. Разрешают повторные посещения
    4. Все переходы возможны (не нулевые)
    """
    # Валидация входных данных
    if type(input_data) != "dict" or "matrix" not in input_data:
        return ("error", "Invalid input format")

    matrix = input_data["matrix"]
    num_cities = len(matrix)

    if not all([len(row) == num_cities for row in matrix]) or num_cities < 2:
        return ("error", "Invalid matrix")

    start_city = 0
    max_depth = num_cities * 2  # Максимальная длина маршрута

    # Инициализация отслеживания посещений
    def create_visited():
        return {city: (city == start_city) for city in range(num_cities)}

    # Генерация маршрутов
    all_routes = []

    # Первый уровень - стартовый город
    current_level = [{
        "path": [start_city],
        "visited": create_visited()
    }]

    # Обход в ширину с ограничением глубины
    for _ in range(max_depth):
        next_level = []
        for route in current_level:
            last_city = route["path"][-1]

            for next_city in range(num_cities):
                if matrix[last_city][next_city] == 0:
                    continue

                new_path = route["path"] + [next_city]

                # Ручное копирование словаря
                new_visited = {}
                for city, status in route["visited"].items():
                    new_visited[city] = status
                new_visited[next_city] = True

                # Проверка завершения маршрута
                if next_city == start_city and all(new_visited.values()):
                    all_routes.append(new_path)

                if len(new_path) < max_depth:
                    next_level.append({
                        "path": new_path,
                        "visited": new_visited
                    })

        current_level = next_level

    # Пагинация
    total = len(all_routes)
    if start >= total:
        return ("empty", None)

    end = start + amount
    if end > total:
        end = total

    return ("ok", {
        "matrix": matrix,
        "routes": all_routes[start:end]
    })