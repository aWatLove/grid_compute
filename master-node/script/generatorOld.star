def generate(input_data, amount, start):
    """
    Генерирует возможные гамильтоновы циклы:
    - Посещает все города ровно 1 раз
    - Возвращается в стартовый город
    - Все переходы возможны (не нулевые)
    """
    # Валидация входных данных
    if not isinstance(input_data, dict) or "matrix" not in input_data:
        return ("error", "Invalid input format")

    matrix = input_data["matrix"]
    num_cities = len(matrix)

    if not all(len(row) == num_cities for row in matrix) or num_cities < 2:
        return ("error", "Invalid matrix")

    # Реализация permutations без itertools
    def permutations(elements):
        if len(elements) <= 1:
            yield elements
        else:
            for perm in permutations(elements[1:]):
                for i in range(len(elements)):
                    yield perm[:i] + elements[0:1] + perm[i:]

    # Фиксируем стартовый город
    start_city = 0
    required_cities = set(range(num_cities))

    # Генератор допустимых маршрутов
    def valid_routes():
        for perm in permutations(list(range(1, num_cities))):
            current_city = start_city
            route = [current_city]
            visited = {current_city}
            valid = True

            # Проверка пути
            for next_city in perm:
                if matrix[current_city][next_city] == 0 or next_city in visited:
                    valid = False
                    break
                route.append(next_city)
                visited.add(next_city)
                current_city = next_city

            # Проверка возврата и полного охвата
            if valid and matrix[current_city][start_city] != 0 and len(visited) == num_cities:
                route.append(start_city)
                yield route

    # Пагинация без itertools.islice
    try:
        gen = valid_routes()
        selected = []
        skipped = 0

        # Пропускаем начальные элементы
        while skipped < start:
            try:
                next(gen)
                skipped += 1
            except StopIteration:
                return ("empty", None)

        # Собираем нужное количество
        for _ in range(amount):
            selected.append(next(gen))
    except StopIteration:
        pass  # Закончились маршруты

    return ("ok", {"matrix": matrix, "routes": selected}) if selected else ("empty", None)