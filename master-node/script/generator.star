def generate(input_data, amount, start):
    """
    Генерирует возможные гамильтоновы циклы:
    - Посещает все города ровно 1 раз
    - Возвращается в стартовый город
    - Все переходы возможны (не нулевые)
    """
    # Валидация входных данных (Starlark-совместимая проверка типа)
    if type(input_data) != "dict" or "matrix" not in input_data:
        return ("error", "Invalid input format")

    matrix = input_data["matrix"]
    num_cities = len(matrix)

    # Проверка матрицы
    matrix_check = []
    for row in matrix:
        matrix_check.append(len(row) == num_cities)

    if not all(matrix_check) or num_cities < 2:
        return ("error", "Invalid matrix")

    # Реализация permutations без рекурсии (Starlark-совместимая)
    def permutations(elements):
        result = [[]]
        for elem in elements:
            new_permutations = []
            for perm in result:
                for i in range(len(perm) + 1):
                    new_permutations.append(perm[:i] + [elem] + perm[i:])
            result = new_permutations
        return result

    # Фиксируем стартовый город
    start_city = 0

    # Генерация маршрутов
    all_routes = []
    for perm in permutations(list(range(1, num_cities))):
        current_city = start_city
        route = [current_city]
        valid = True

        for next_city in perm:
            if matrix[current_city][next_city] == 0:
                valid = False
                break
            route.append(next_city)
            current_city = next_city

        if valid and matrix[current_city][start_city] != 0:
            route.append(start_city)
            all_routes.append(route)

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