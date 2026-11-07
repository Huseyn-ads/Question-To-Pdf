/**
 * DEMO DATA — fictional content for the phase-1 design preview only.
 * Never used by the real application routes.
 */
import type { QuestionBank, QuizSummary, User } from '@/api/types'

export const demoUser: User = {
  id: 'demo-user',
  email: 'student@example.test',
  name: 'Айсель',
  created_at: '2026-09-01T09:00:00Z',
  email_verified_at: '2026-09-01T09:05:00Z',
}

const daysAgo = (days: number, hours = 0) => new Date(Date.now() - days * 86_400_000 - hours * 3_600_000).toISOString()

export const demoBanks = (): QuestionBank[] => [
  {
    id: 'bank-algo',
    title: 'Алгоритмы и программирование — экзамен 2026',
    description: 'Импорт из DOCX «Alq və prq 325 test imtahan 2026»',
    created_at: daysAgo(9),
    updated_at: daysAgo(0, 3),
  },
  {
    id: 'bank-phil',
    title: 'Философия — коллоквиум',
    description: '1 семестр, темы 1–8',
    created_at: daysAgo(14),
    updated_at: daysAgo(1, 2),
  },
  {
    id: 'bank-arch',
    title: 'Kompüter arxitekturası — 150 sual',
    description: 'Ümumi kollokvium sualları',
    created_at: daysAgo(20),
    updated_at: daysAgo(4),
  },
  {
    id: 'bank-iak',
    title:
      'Информационно-аналитические компьютерные системы: второй коллоквиум для студентов дневного отделения, весенний семестр 2026 года',
    description: '',
    created_at: daysAgo(30),
    updated_at: daysAgo(12),
  },
  {
    id: 'bank-soft',
    title: 'Yumşaq bacarıqlar (Kollokvium)',
    description: 'Ünsiyyət, komanda işi və vaxtın idarə olunması',
    created_at: daysAgo(40),
    updated_at: daysAgo(25),
  },
]

export const demoQuizzes = (): QuizSummary[] => [
  {
    id: 'quiz-algo',
    question_bank_id: 'bank-algo',
    question_count: 12,
    time_limit_minutes: 25,
    shuffle_options: true,
    created_at: daysAgo(0, 2),
  },
  {
    id: 'quiz-phil',
    question_bank_id: 'bank-phil',
    question_count: 10,
    time_limit_minutes: 0,
    shuffle_options: false,
    created_at: daysAgo(1, 5),
  },
  {
    id: 'quiz-arch',
    question_bank_id: 'bank-arch',
    question_count: 12,
    time_limit_minutes: 15,
    shuffle_options: true,
    created_at: daysAgo(4, 1),
  },
  {
    id: 'quiz-iak',
    question_bank_id: 'bank-iak',
    question_count: 8,
    time_limit_minutes: 60,
    shuffle_options: true,
    created_at: daysAgo(12),
  },
]

export interface DemoQuestion {
  text: string
  options: string[]
  correct: number
}

/** Question pool: long text, code with line breaks, Unicode, 8 options. */
export const demoQuestionPool: DemoQuestion[] = [
  {
    text: 'Какова временная сложность бинарного поиска в отсортированном массиве из n элементов?',
    options: ['O(n)', 'O(log n)', 'O(n log n)', 'O(1)'],
    correct: 1,
  },
  {
    text: 'Студент реализует очередь с приоритетом для планировщика задач операционной системы. Задачи поступают непрерывно, у каждой есть числовой приоритет; планировщик должен быстро извлекать задачу с наивысшим приоритетом и так же быстро добавлять новые. Какая структура данных подходит лучше всего, если важны обе операции, а объём памяти ограничен?',
    options: [
      'Отсортированный односвязный список',
      'Двоичная куча, хранящаяся в массиве',
      'Хеш-таблица с цепочками',
      'Неотсортированный динамический массив',
    ],
    correct: 1,
  },
  {
    text: "Что выведет следующий фрагмент программы?\n\nfor i in range(3):\n    print(i * 2, end=' ')",
    options: ['0 2 4', '2 4 6', '0 1 2', 'Ошибка выполнения'],
    correct: 0,
  },
  {
    text: 'Alqoritmin əsas xüsusiyyətlərinə hansı aid deyil?',
    options: ['Diskretlik', 'Müəyyənlik', 'Kütləvilik', 'Qeyri-müəyyənlik', 'Nəticəlilik'],
    correct: 3,
  },
  {
    text: 'Какой из вариантов перечисляет только устойчивые алгоритмы сортировки?',
    options: [
      'Быстрая сортировка, пирамидальная сортировка',
      'Сортировка слиянием, сортировка вставками',
      'Сортировка выбором, быстрая сортировка',
      'Пирамидальная сортировка, сортировка Шелла',
      'Сортировка Шелла, сортировка выбором',
      'Быстрая сортировка, сортировка слиянием',
      'Сортировка подсчётом, пирамидальная сортировка',
      'Introsort, сортировка выбором',
    ],
    correct: 1,
  },
  {
    text: 'Что такое рекурсия?',
    options: [
      'Способ описания функции, при котором функция прямо или косвенно вызывает саму себя, а условие выхода гарантирует завершение вычислений',
      'Повторение блока команд заранее известное число раз',
      'Хранение промежуточных результатов вычислений в кэше процессора',
      'Параллельное выполнение нескольких копий одной функции',
    ],
    correct: 0,
  },
  {
    text: 'Какой обход двоичного дерева поиска выдаёт ключи в порядке возрастания?',
    options: ['Прямой (pre-order)', 'Симметричный (in-order)', 'Обратный (post-order)', 'В ширину'],
    correct: 1,
  },
  {
    text: 'Сколько рёбер в связном дереве с 12 вершинами?',
    options: ['10', '11', '12', '13'],
    correct: 1,
  },
  {
    text: 'Какая нотация описывает асимптотическую верхнюю оценку?',
    options: ['Θ (тета)', 'Ω (омега)', 'O (о большое)', 'ω (омега малое)'],
    correct: 2,
  },
  {
    text: 'Hansı məlumat strukturu LIFO prinsipi ilə işləyir?',
    options: ['Növbə', 'Stek', 'Massiv', 'Ağac'],
    correct: 1,
  },
  {
    text: 'Для какой задачи предназначен алгоритм Дейкстры?',
    options: [
      'Поиск кратчайших путей от одной вершины во взвешенном графе с неотрицательными весами рёбер',
      'Построение минимального остовного дерева',
      'Топологическая сортировка ориентированного ациклического графа',
      'Поиск максимального потока в сети',
    ],
    correct: 0,
  },
  {
    text: 'Какое утверждение о хеш-таблицах с открытой адресацией верно?',
    options: [
      'Коллизии разрешаются хранением списков в каждой ячейке',
      'При высоком коэффициенте заполнения время поиска заметно растёт',
      'Удаление элемента не требует специальных пометок',
      'Размер таблицы не влияет на производительность',
    ],
    correct: 1,
  },
]
