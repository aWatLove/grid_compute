def generate(input_data, amount, start):
    """
    Генерирует маршруты для всех стартовых городов по очереди:
    1. Посещает ВСЕ города минимум 1 раз
    2. Возвращается в текущий стартовый город
    3. Разрешает повторные посещения
    4. Все переходы возможны (не нулевые)
    """
    # Валидация входных данных
    if type(input_data) != "dict" or "matrix" not in input_data:
        return ("error", "Invalid input format")

    matrix = input_data["matrix"]
    num_cities = len(matrix)

    if not all([len(row) == num_cities for row in matrix]) or num_cities < 2:
        return ("error", "Invalid matrix")

    max_depth = num_cities * 2
    all_routes = []

    # Перебираем все возможные стартовые города
    for start_city in range(num_cities):
        # Инициализация отслеживания посещений для текущего стартового города
        def create_visited():
            return {city: (city == start_city) for city in range(num_cities)}

        # Генерация маршрутов для текущего стартового города
        current_level = [{
            "path": [start_city],
            "visited": create_visited()
        }]

        # Обход в ширину для текущего стартового города
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

    # Пагинация с учетом всех стартовых городов
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